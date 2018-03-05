// Copyright 2015-2018 trivago N.V.
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
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
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

// AwsMultiClient component
//
// The AwsMultiClient is a helper component to handle aws access and client instantiation
//
// Parameters
//
// - Region: This value defines the used aws region.
// By default this is set to "us-east-1"
//
// - Endpoint: This value defines the used aws api endpoint. If no endpoint is set
// the client needs to set the right endpoint for the used region.
// By default this is set to "".
//
type AwsMultiClient struct {
	Credentials AwsCredentials `gollumdoc:"embed_type"`
	region      string         `config:"Region" default:"us-east-1"`
	endpoint    string         `config:"Endpoint" default:""`
	config      *aws.Config
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
	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            *client.config,
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		return nil, err
	}

	if client.Credentials.assumeRole != "" {
		credentials := stscreds.NewCredentials(sess, client.Credentials.assumeRole)
		client.config.WithCredentials(credentials)
	}

	return sess, err
}

// AwsCredentials is a config struct for aws credential handling
//
// Parameters
//
// - Credential/Type: This value defines the credentials that are to be used when
// connecting to aws. Available values are listed below. See
// https://docs.aws.amazon.com/sdk-for-go/api/aws/credentials/#Credentials
// for more information.
//  - environment: Retrieves credentials from the environment variables of
//  the running process
//  - static: Retrieves credentials value for individual credential fields
//  - shared: Retrieves credentials from the current user's home directory
//  - none: Use a anonymous login to aws
//
// - Credential/Id: is used for "static" type and is used as the AccessKeyID
//
// - Credential/Token: is used for "static" type and is used as the SessionToken
//
// - Credential/Secret: is used for "static" type and is used as the SecretAccessKey
//
// - Credential/File: is used for "shared" type and is used as the path to your
// shared Credentials file (~/.aws/credentials)
//
// - Credential/Profile: is used for "shared" type and is used for the profile
//
// - Credential/AssumeRole: This value is used to assume an IAM role using.
// By default this is set to "".
//
type AwsCredentials struct {
	credentialType string `config:"Credential/Type" default:"none"`
	staticID       string `config:"Credential/Id" default:""`
	staticToken    string `config:"Credential/Token" default:""`
	staticSecret   string `config:"Credential/Secret" default:""`
	sharedFile     string `config:"Credential/File" default:""`
	sharedProfile  string `config:"Credential/Profile" default:"default"`
	assumeRole     string `config:"Credential/AssumeRole" default:""`
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
