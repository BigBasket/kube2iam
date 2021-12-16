package iam

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/jtblin/kube2iam/metrics"
	"github.com/sirupsen/logrus"
)

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
	name := fmt.Sprintf("%s-%s-%s", getHash(remoteIP), roleARN[idx+1:], strconv.Itoa(rand.Int()))
	sessionName := fmt.Sprintf("%.[2]*[1]s", name, maxSessNameLength)
	logrus.Debugf("session for the role: %v remoteip: %v session: %v", roleARN, remoteIP, sessionName)

	return sessionName
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
	// Set up a prometheus timer to track the AWS request duration. It stores the timer value when
	// observed. A function gets err at observation time to report the status of the request after the function returns.

	var assumeRoleOutput *sts.AssumeRoleOutput
	var assumeRoleOutputError error

	assumeRoleInput := sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName(roleARN, remoteIP)),
	}
	// Only inject the externalID if one was provided with the request
	if externalID != "" {
		assumeRoleInput.ExternalId = &externalID
	}

	cfg, _ := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithClientLogMode(aws.LogRequest|aws.LogResponse|aws.LogRetries),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 0)
		}),
	)

	if iam.UseRegionalEndpoint {
		cfg.EndpointResolverWithOptions = iam
	}
	cfg.HTTPClient = &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	stsClient := sts.NewFromConfig(cfg)
	aStart := time.Now()
	assumeRoleOutput, assumeRoleOutputError = stsClient.AssumeRole(context.TODO(), &assumeRoleInput)
	logrus.Infof("time taken to complete the assumerole: %v", time.Since(aStart).Milliseconds())

	if assumeRoleOutputError != nil {
		logrus.Error(assumeRoleOutputError)

		return nil, assumeRoleOutputError
	}

	return &Credentials{
		AccessKeyID:     *assumeRoleOutput.Credentials.AccessKeyId,
		Code:            "Success",
		Expiration:      assumeRoleOutput.Credentials.Expiration.Format("2006-01-02T15:04:05Z"),
		LastUpdated:     time.Now().Format("2006-01-02T15:04:05Z"),
		SecretAccessKey: *assumeRoleOutput.Credentials.SecretAccessKey,
		Token:           *assumeRoleOutput.Credentials.SessionToken,
		Type:            "AWS-HMAC",
	}, nil
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
