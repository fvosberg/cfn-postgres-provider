package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/fvosberg/cfn-postgres-provider/internal"
	"github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

// TODO delete resource request?
//
// TODO add tracing
// TODO README for deployment via cloudfront
// TODO README for deployment via aws cli
// TODO README for manual deployment in AWS management console
// TODO add local end to end tests

var (
	log     = logrus.New()
	service *internal.Service
)

func init() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.DebugLevel)

	sess := session.New()
	service = internal.NewService(
		log,
		ssm.New(sess),
	)
}

func main() {
	lambda.Start(cfn.LambdaWrap(handleLambda))
}

func handleLambda(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	logEvent := event
	log.WithField("event", logEvent).Info("Triggered by new event")

	switch event.RequestType {
	case cfn.RequestCreate:
		fallthrough
	case cfn.RequestUpdate:
		id, err := service.CreateDBWithOwner(ctx, event.ResourceProperties)
		return id, nil, err
	case cfn.RequestDelete:
		// TODO implement delete by physicalResourceID
		return event.PhysicalResourceID, nil, nil
	}

	return "", nil, fmt.Errorf("unsupported request type %q", event.RequestType)
}
