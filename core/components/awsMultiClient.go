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
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/trivago/gollum/core"
)

//TODO: check and handle endpoint for other aws services
const (
	defaultEndpoint = "s3.amazonaws.com"
	defaultRegion   = "us-east-1"
)

const (
	CredentialTypeEnv    = "environment"
	CredentialTypeStatic = "static"
	CredentialTypeShared = "shared"
	CredentialTypeNone   = "none"
)

type AwsMultiClient struct {
	Credentials AwsCredentials

	region   string `config:"Region" default:"us-east-1"`
	endpoint string `config:"Endpoint" default:""`

	config *aws.Config
}

func (client *AwsMultiClient) Configure(conf core.PluginConfigReader) {
	if client.endpoint == "" {
		if client.region != defaultRegion {
			client.endpoint = fmt.Sprintf("s3-%s.amazonaws.com", client.region)
		} else {
			client.endpoint = defaultEndpoint
		}
	}

	client.config = aws.NewConfig()

	client.config.WithRegion(client.region)
	client.config.WithEndpoint(client.endpoint)

	client.config.CredentialsChainVerboseErrors = aws.Bool(true)
	client.config.WithCredentials(client.Credentials.CreateAwsCredentials())
}

func (client *AwsMultiClient) GetS3Client() (*s3.S3, error) {
	sess, err := client.newSessionWithOptions()
	if err != nil {
		return nil, err
	}

	return s3.New(sess, client.config), nil
}

func (client *AwsMultiClient) newSessionWithOptions() (*session.Session, error) {
	return session.NewSessionWithOptions(session.Options{
		Config:            *client.config,
		SharedConfigState: session.SharedConfigEnable,
	})
}

type AwsCredentials struct {
	credentialType string `config:"Credential/Type" default:"none"`
	staticID       string `config:"Credential/Id" default:""`
	staticToken    string `config:"Credential/Token" default:""`
	staticSecret   string `config:"Credential/Secret" default:""`
	sharedFile     string `config:"Credential/File" default:""`
	sharedProfile  string `config:"Credential/Profile" default:"default"`
}

func (cred *AwsCredentials) CreateAwsCredentials() *credentials.Credentials {
	switch cred.credentialType {
	case CredentialTypeEnv:
		return credentials.NewEnvCredentials()

	case CredentialTypeStatic:
		return credentials.NewStaticCredentials(cred.staticID, cred.staticSecret, cred.staticToken)

	case CredentialTypeShared:
		return credentials.NewSharedCredentials(cred.sharedFile, cred.sharedProfile)

	case CredentialTypeNone:
		return credentials.AnonymousCredentials

	default:
		panic(fmt.Sprintf("Unknown CredentialType: %s", cred.credentialType))
	}
}
