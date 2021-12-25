package gamelift

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/gamelift"
	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
)

var logger = logutil.GetLogger()

type Client struct {
	ctx    *context.Context
	client *gamelift.Client
}

func NewAwsClient() *Client {
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL: "http://localhost:9080",
				}, nil
			})))
	if err != nil {
		logger.Fatalf("unable to load SDK config, %v", err)
	}

	// Using the Config value, create the GameLift client
	return &Client {
		&ctx,
		gamelift.NewFromConfig(cfg),
	}
}

func (client *Client) CountAvailableGameSessions(fleetId string) (int, error) {
	params := &gamelift.DescribeGameSessionsInput{
		FleetId: &fleetId,
	}

	result, err := client.client.DescribeGameSessions(*client.ctx, params)
	if err != nil {
		return 0, err
	}

	availableSessionCount := 0
	for _, gameSession := range result.GameSessions {
		if gameSession.Status == "ACTIVATING" || gameSession.Status == "ACTIVE" {
			availableSessionCount++
		}
	}

	return availableSessionCount, nil
}

func (client *Client) GetPercentAvailableGameSessions(fleetId string) (float32, error) {
	// TODO(wj): Current show do mock!!
	totalProcessNum := 4
	availableGameSessionNum, err := client.CountAvailableGameSessions(fleetId)
	if err != nil {
		return 0, err
	}
	return 1 - float32(availableGameSessionNum) / float32(totalProcessNum), nil
}
