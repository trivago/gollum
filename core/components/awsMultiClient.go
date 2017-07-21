// Copyright 2015-2017 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package components

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/trivago/gollum/core"
)

// DefaultAwsRegion defines the default region to use
const DefaultAwsRegion = "us-east-1"

const (
	credentialTypeEnv    = "environment"
	credentialTypeStatic = "static"
	credentialTypeShared = "shared"
	credentialTypeNone   = "none"
)

// AwsMultiClient is a helper component to handle aws access and client instantiation
//
// Region defines the used aws region. By default this is set to "us-east-1"
//
// Endpoint defines the used aws api endpoint. By default this is set to "" and the client
// tries to get the right endpoint for the used region.
//
type AwsMultiClient struct {
	Credentials AwsCredentials

	region   string `config:"Region" default:"us-east-1"`
	endpoint string `config:"Endpoint" default:""`

	config *aws.Config
}

// Configure method
func (client *AwsMultiClient) Configure(conf core.PluginConfigReader) {
	client.config = aws.NewConfig()

	client.config.WithRegion(client.region)
	client.config.WithEndpoint(client.endpoint)

	credentials, err := client.Credentials.CreateAwsCredentials()
	if err != nil {
		conf.Errors.Push(err)
	}

	client.config.WithCredentials(credentials)
	client.config.CredentialsChainVerboseErrors = aws.Bool(true)
}

// GetConfig returns set *aws.Config
func (client *AwsMultiClient) GetConfig() *aws.Config {
	return client.config
}

// NewSessionWithOptions returns a instantiated asw session
func (client *AwsMultiClient) NewSessionWithOptions() (*session.Session, error) {
	return session.NewSessionWithOptions(session.Options{
		Config:            *client.config,
		SharedConfigState: session.SharedConfigEnable,
	})
}

// AwsCredentials is a config struct for aws credential handling
//
// Credential/Type defines the credentials that are to be used when
// connecting to s3. This can be one of the following: "environment",
// "static", "shared" and "none". By default this is set to "none".
// See https://docs.aws.amazon.com/sdk-for-go/api/aws/credentials/#Credentials for more information
//
// Credential/Id is used for "static" type and is used as the AccessKeyID
//
// Credential/Token is used for "static" type and is used as the SessionToken
//
// Credential/Secret is used for "static" type and is used as the SecretAccessKey
//
// Credential/File is used for "shared" type and is used as the path to your
// shared Credentials file (~/.aws/credentials)
//
// Credential/Profile is used for "shared" type and is used for the profile
//
type AwsCredentials struct {
	credentialType string `config:"Credential/Type" default:"none"`
	staticID       string `config:"Credential/Id" default:""`
	staticToken    string `config:"Credential/Token" default:""`
	staticSecret   string `config:"Credential/Secret" default:""`
	sharedFile     string `config:"Credential/File" default:""`
	sharedProfile  string `config:"Credential/Profile" default:"default"`
}

// CreateAwsCredentials returns aws credentials.Credentials for active settings
func (cred *AwsCredentials) CreateAwsCredentials() (*credentials.Credentials, error) {
	switch cred.credentialType {
	case credentialTypeEnv:
		return credentials.NewEnvCredentials(), nil

	case credentialTypeStatic:
		return credentials.NewStaticCredentials(cred.staticID, cred.staticSecret, cred.staticToken), nil

	case credentialTypeShared:
		return credentials.NewSharedCredentials(cred.sharedFile, cred.sharedProfile), nil

	case credentialTypeNone:
		return credentials.AnonymousCredentials, nil

	default:
		return credentials.AnonymousCredentials, fmt.Errorf("Unknown CredentialType: %s", cred.credentialType)
	}
}
