package iam

import (
	"fmt"
	"regexp"
	"strings"
)

const fullArnPrefix = "arn:"

// ARNRegexp is the regex to check that the base ARN is valid,
// see http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html#identifiers-arns.
var ARNRegexp = regexp.MustCompile(`^arn:(\w|-)*:iam::\d+:role\/?(\w+|-|\/|\.)*$`)

// IsValidBaseARN validates that the base ARN is valid.
func IsValidBaseARN(arn string) bool {
	return ARNRegexp.MatchString(arn)
}

// RoleARN returns the full iam role ARN.
func (iam *Client) RoleARN(role string) string {
	if strings.HasPrefix(strings.ToLower(role), fullArnPrefix) {
		return role
	}
	return fmt.Sprintf("%s%s", iam.BaseARN, role)
}

// GetBaseArn get the base ARN from metadata service.
func GetBaseArn() (string, error) {
	// base arn is already provided as the command line argument

	return "", nil
}
