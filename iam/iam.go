package iam

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/jtblin/kube2iam/metrics"
	"github.com/karlseguin/ccache"
	"github.com/sirupsen/logrus"
)

var cache = ccache.New(ccache.Configure())

const (
	maxSessNameLength = 64
)

// Client represents an IAM client.
type Client struct {
	BaseARN             string
	Endpoint            string
	UseRegionalEndpoint bool
	StsVpcEndPoint      string
	StsClient           *sts.Client
}

// Credentials represent the security Credentials response.
type Credentials struct {
	AccessKeyID     string `json:"AccessKeyId"`
	Code            string
	Expiration      string
	LastUpdated     string
	SecretAccessKey string
	Token           string
	Type            string
}

func getHash(text string) string {
	h := fnv.New32a()
	_, err := h.Write([]byte(text))
	if err != nil {
		return text
	}
	return fmt.Sprintf("%x", h.Sum32())
}

func GetInstanceIAMRole() (string, error) {
	// instance iam role is already supplied through command line arguments

	return "", nil
}

func sessionName(roleARN, remoteIP string) string {
	idx := strings.LastIndex(roleARN, "/")
	name := fmt.Sprintf("%s-%s", getHash(remoteIP), roleARN[idx+1:])
	return fmt.Sprintf("%.[2]*[1]s", name, maxSessNameLength)
}

// Helper to format IAM return codes for metric labeling
func getIAMCode(err error) string {
	if err != nil {
		return metrics.IamUnknownFailCode
	}

	return metrics.IamSuccessCode
}

// GetEndpointFromRegion formas a standard sts endpoint url given a region
func GetEndpointFromRegion(region string) string {
	endpoint := fmt.Sprintf("https://sts.%s.amazonaws.com", region)
	if strings.HasPrefix(region, "cn-") {
		endpoint = fmt.Sprintf("https://sts.%s.amazonaws.com.cn", region)
	}
	return endpoint
}

func IsValidRegion(region string) bool {
	return true
}

func (iam *Client) ResolveEndpoint(service, region string, options ...interface{}) (aws.Endpoint, error) {
	if service == "sts" {
		if iam.StsVpcEndPoint == "" {
			iam.Endpoint = GetEndpointFromRegion(region)
		} else {
			iam.Endpoint = iam.StsVpcEndPoint
		}
		return aws.Endpoint{
			URL:           iam.Endpoint,
			SigningRegion: region,
		}, nil
	}

	return aws.Endpoint{}, nil
}

// AssumeRole returns an IAM role Credentials using AWS STS.
func (iam *Client) AssumeRole(roleARN, externalID string, remoteIP string, sessionTTL time.Duration) (*Credentials, error) {
	hitCache := true
	item, err := cache.Fetch(roleARN, 0*time.Second, func() (interface{}, error) {
		hitCache = false

		// Set up a prometheus timer to track the AWS request duration. It stores the timer value when
		// observed. A function gets err at observation time to report the status of the request after the function returns.
		var err error
		lvsProducer := func() []string {
			return []string{getIAMCode(err), roleARN}
		}
		timer := metrics.NewFunctionTimer(metrics.IamRequestSec, lvsProducer, nil)
		defer timer.ObserveDuration()

		assumeRoleInput := sts.AssumeRoleInput{
			DurationSeconds: aws.Int32(int32(sessionTTL.Seconds() * 2)),
			RoleArn:         aws.String(roleARN),
			RoleSessionName: aws.String(sessionName(roleARN, remoteIP)),
		}
		// Only inject the externalID if one was provided with the request
		if externalID != "" {
			assumeRoleInput.ExternalId = &externalID
		}
		resp, err := iam.StsClient.AssumeRole(context.TODO(), &assumeRoleInput)
		if err != nil {
			logrus.Error(err)

			return nil, err
		}

		return &Credentials{
			AccessKeyID:     *resp.Credentials.AccessKeyId,
			Code:            "Success",
			Expiration:      resp.Credentials.Expiration.Format("2006-01-02T15:04:05Z"),
			LastUpdated:     time.Now().Format("2006-01-02T15:04:05Z"),
			SecretAccessKey: *resp.Credentials.SecretAccessKey,
			Token:           *resp.Credentials.SessionToken,
			Type:            "AWS-HMAC",
		}, nil
	})
	if hitCache {
		metrics.IamCacheHitCount.WithLabelValues(roleARN).Inc()
	}
	if err != nil {
		logrus.Error(err)

		return nil, err
	}
	return item.Value().(*Credentials), nil
}

// NewClient returns a new IAM client.
func NewClient(baseARN string, regional bool, stsVpcEndPoint string) (*Client, error) {
	client := &Client{
		BaseARN:             baseARN,
		Endpoint:            "sts.amazonaws.com",
		UseRegionalEndpoint: regional,
		StsVpcEndPoint:      stsVpcEndPoint,
	}

	cfg, cErr := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithClientLogMode(aws.LogRequest|aws.LogResponse|aws.LogRetries))
	if cErr != nil {
		return nil, cErr
	}
	if client.UseRegionalEndpoint {
		cfg.EndpointResolverWithOptions = client
	}
	client.StsClient = sts.NewFromConfig(cfg)

	return client, nil
}
