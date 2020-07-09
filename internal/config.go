package internal

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/mitchellh/mapstructure"
)

type ssmClient struct {
	ssm ssmAgent
}

//go:generate moq -out ssm_agent_moq_test.go . ssmAgent
type ssmAgent interface {
	GetParameterWithContext(aws.Context, *ssm.GetParameterInput, ...request.Option) (*ssm.GetParameterOutput, error)
}

func (a *ssmClient) loadConfigParameters(ctx context.Context, inputConfig config) (Config, error) {
	userPassword, err := a.secretValue(ctx, inputConfig.User.PasswordParameter)
	if err != nil {
		return Config{}, fmt.Errorf(
			"loading parameter %q for the password for the new user failed: %w",
			inputConfig.User.PasswordParameter,
			err,
		)
	}
	connectionPassword, err := a.secretValue(ctx, inputConfig.Connection.PasswordParameter)
	if err != nil {
		return Config{}, fmt.Errorf(
			"loading parameter %q for the super user failed: %w",
			inputConfig.Connection.PasswordParameter,
			err,
		)
	}

	// we are using the implicit syntax without field names intentionally
	// to crash if new fields are added
	res := Config{
		inputConfig.DatabaseName,
		UserConfig{
			inputConfig.User.Name,
			userPassword,
		},
		ConnectionConfig{
			inputConfig.Connection.Host,
			inputConfig.Connection.Port,
			inputConfig.Connection.User,
			connectionPassword,
		},
		inputConfig.Extensions,
	}

	return res, nil
}

func (a *ssmClient) secretValue(ctx context.Context, key string) (string, error) {
	param, err := a.ssm.GetParameterWithContext(ctx, &ssm.GetParameterInput{
		Name:           &key,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	return *param.Parameter.Value, nil
}

type Config struct {
	DatabaseName string
	User         UserConfig

	Connection ConnectionConfig
	Extensions []string
}

type ConnectionConfig struct {
	Host     string
	Port     uint16
	User     string
	Password string
}

type UserConfig struct {
	Name     string
	Password string
}

func decodeConfig(resourceProperties map[string]interface{}) (config, error) {
	var c config
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Result:           &c,
	})
	if err != nil {
		return c, fmt.Errorf("init decoder: %w", err)
	}
	return c, d.Decode(resourceProperties)
}

type config struct {
	ServiceToken string
	Connection   connectionConfig

	DatabaseName string
	User         userConfig
	Extensions   []string
}

type connectionConfig struct {
	Host              string
	Port              uint16
	User              string
	PasswordParameter string
}

type userConfig struct {
	Name              string
	PasswordParameter string
}
