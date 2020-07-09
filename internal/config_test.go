package internal

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func TestDecodeConfigSuccess(t *testing.T) {
	tests := []struct {
		name               string
		resourceProperties map[string]interface{}
		expectedConfig     config
	}{
		{
			name: "valid config",
			resourceProperties: map[string]interface{}{
				"ServiceToken": interface{}("arn:lambda"),
				"DatabaseName": interface{}("theDatabaseToCreate"),
				"User": interface{}(map[string]interface{}{
					"Name":              interface{}("newUser"),
					"PasswordParameter": interface{}("/params/newUser/password"),
				}),
				"Connection": interface{}(map[string]interface{}{
					"Host":              interface{}("db.internal.host"),
					"Port":              interface{}(5432),
					"User":              interface{}("root"),
					"PasswordParameter": interface{}("/params/root/password"),
				}),
				"Extensions": interface{}([]string{"uuid-ossp"}),
			},
			expectedConfig: config{
				ServiceToken: "arn:lambda",
				DatabaseName: "theDatabaseToCreate",
				User: userConfig{
					Name:              "newUser",
					PasswordParameter: "/params/newUser/password",
				},
				Connection: connectionConfig{
					Host:              "db.internal.host",
					Port:              5432,
					User:              "root",
					PasswordParameter: "/params/root/password",
				},
				Extensions: []string{"uuid-ossp"},
			},
		},
		{
			name: "port provided as string",
			resourceProperties: map[string]interface{}{
				"ServiceToken": interface{}("arn:lambda"),
				"DatabaseName": interface{}("theDatabaseToCreate"),
				"User": interface{}(map[string]interface{}{
					"Name":              interface{}("newUser"),
					"PasswordParameter": interface{}("/params/newUser/password"),
				}),
				"Connection": interface{}(map[string]interface{}{
					"Host":              interface{}("db.internal.host"),
					"Port":              interface{}("5432"),
					"User":              interface{}("root"),
					"PasswordParameter": interface{}("/params/root/password"),
				}),
				"Extensions": interface{}([]string{"uuid-ossp"}),
			},
			expectedConfig: config{
				ServiceToken: "arn:lambda",
				DatabaseName: "theDatabaseToCreate",
				User: userConfig{
					Name:              "newUser",
					PasswordParameter: "/params/newUser/password",
				},
				Connection: connectionConfig{
					Host:              "db.internal.host",
					Port:              5432,
					User:              "root",
					PasswordParameter: "/params/root/password",
				},
				Extensions: []string{"uuid-ossp"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := decodeConfig(tt.resourceProperties)
			if err != nil {
				t.Fatalf("Unexpected error decoding: %s", err)
			}
			if !reflect.DeepEqual(cfg, tt.expectedConfig) {
				t.Errorf(
					"Unexpected config\nactual:\t%#v\nexpect:\t%#v",
					cfg,
					tt.expectedConfig,
				)
			}
		})
	}
}

func TestLoadConfigParameters(t *testing.T) {
	tests := []struct {
		name           string
		ssmMock        *ssmAgentMock
		inputConfig    config
		expectedConfig Config
	}{
		{
			name: "valid config",
			ssmMock: &ssmAgentMock{
				GetParameterWithContextFunc: func(
					ctx context.Context,
					input *ssm.GetParameterInput,
					opts ...request.Option,
				) (*ssm.GetParameterOutput, error) {
					if *input.Name == "/params/newUser/password" {
						return &ssm.GetParameterOutput{
							Parameter: &ssm.Parameter{
								Value: aws.String("newUserPassword"),
							},
						}, nil
					}
					if *input.Name == "/params/root/password" {
						return &ssm.GetParameterOutput{
							Parameter: &ssm.Parameter{
								Value: aws.String("rootPassword"),
							},
						}, nil
					}
					return nil, errors.New("unexpected parameter")
				},
			},
			inputConfig: config{
				ServiceToken: "arn:lambda",
				DatabaseName: "theDatabaseToCreate",
				User: userConfig{
					Name:              "newUser",
					PasswordParameter: "/params/newUser/password",
				},
				Connection: connectionConfig{
					Host:              "db.internal.host",
					Port:              5432,
					User:              "root",
					PasswordParameter: "/params/root/password",
				},
			},
			expectedConfig: Config{
				DatabaseName: "theDatabaseToCreate",
				User: UserConfig{
					Name:     "newUser",
					Password: "newUserPassword",
				},
				Connection: ConnectionConfig{
					Host:     "db.internal.host",
					Port:     5432,
					User:     "root",
					Password: "rootPassword",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ssmClient{ssm: tt.ssmMock}

			cfg, err := c.loadConfigParameters(
				context.Background(),
				tt.inputConfig,
			)
			if err != nil {
				t.Fatalf("Unexpected error loading config: %s", err)
			}
			if !reflect.DeepEqual(cfg, tt.expectedConfig) {
				t.Errorf(
					"Unexpected config\nactual:\t%#v\nexpect:\t%#v",
					cfg,
					tt.expectedConfig,
				)
			}
			calls := tt.ssmMock.GetParameterWithContextCalls()
			if len(calls) != 2 {
				t.Fatalf("Unexpected number of calls for ssm client: %d, expected 2", len(calls))
			}
			if *calls[0].In2.Name != "/params/newUser/password" {
				t.Errorf(
					"Expected to get the parameter key %q, but got %q",
					"/params/newUser/password",
					*calls[0].In2.Name,
				)
			}
			if calls[0].In2.WithDecryption == nil || !*calls[1].In2.WithDecryption {
				t.Error("Expected to get the first param with decryption")
			}
			if *calls[1].In2.Name != "/params/root/password" {
				t.Errorf(
					"Expected to get the parameter key %q, but got %q",
					"/params/root/password",
					*calls[0].In2.Name,
				)
			}
			if calls[1].In2.WithDecryption == nil || !*calls[1].In2.WithDecryption {
				t.Error("Expected to get the second param with decryption")
			}
		})
	}
}

// TODO failed loading of config parameters
