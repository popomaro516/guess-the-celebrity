package feed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	feedID               = "random"
	maxQuizzes           = int32(10)
	publishedStatus      = "published"
	statusCreatedAtIndex = "status-created-at-index"
	publicQuizProjection = "quiz_id, question, choices, difficulty, cropped_image_key"
)

var publicQuizFields = []string{
	"quiz_id",
	"question",
	"choices",
	"difficulty",
	"cropped_image_key",
}

type DynamoDBAPI interface {
	Query(
		ctx context.Context,
		params *dynamodb.QueryInput,
		optFns ...func(*dynamodb.Options),
	) (*dynamodb.QueryOutput, error)
	PutItem(
		ctx context.Context,
		params *dynamodb.PutItemInput,
		optFns ...func(*dynamodb.Options),
	) (*dynamodb.PutItemOutput, error)
}

type Config struct {
	QuizzesTableName string
	FeedTableName    string
}

type Result struct {
	QuizCount int `json:"quiz_count"`
}

type Worker struct {
	dynamodb     DynamoDBAPI
	quizzesTable string
	feedTable    string
	now          func() time.Time
}

func New(dynamodbClient DynamoDBAPI, config Config, now func() time.Time) (*Worker, error) {
	if config.QuizzesTableName == "" {
		return nil, errors.New("DYNAMODB_QUIZZES_TABLE_NAME is required")
	}
	if config.FeedTableName == "" {
		return nil, errors.New("DYNAMODB_QUIZ_FEED_TABLE_NAME is required")
	}
	if dynamodbClient == nil {
		return nil, errors.New("DynamoDB client is required")
	}
	if now == nil {
		now = time.Now
	}
	return &Worker{
		dynamodb:     dynamodbClient,
		quizzesTable: config.QuizzesTableName,
		feedTable:    config.FeedTableName,
		now:          now,
	}, nil
}

func (w *Worker) Refresh(ctx context.Context) (Result, error) {
	out, err := w.dynamodb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(w.quizzesTable),
		IndexName:              aws.String(statusCreatedAtIndex),
		KeyConditionExpression: aws.String("#status = :published"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":published": &types.AttributeValueMemberS{Value: publishedStatus},
		},
		ProjectionExpression: aws.String(publicQuizProjection),
		ScanIndexForward:     aws.Bool(false),
		Limit:                aws.Int32(maxQuizzes),
	})
	if err != nil {
		return Result{}, fmt.Errorf("query published quizzes: %w", err)
	}

	quizValues := make([]types.AttributeValue, 0, len(out.Items))
	for index, item := range out.Items {
		if int32(index) == maxQuizzes {
			break
		}
		publicItem, err := publicQuizItem(item)
		if err != nil {
			return Result{}, fmt.Errorf("build public quiz feed item: %w", err)
		}
		quizValues = append(quizValues, &types.AttributeValueMemberM{Value: publicItem})
	}

	_, err = w.dynamodb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(w.feedTable),
		Item: map[string]types.AttributeValue{
			"feed_id":    &types.AttributeValueMemberS{Value: feedID},
			"quizzes":    &types.AttributeValueMemberL{Value: quizValues},
			"updated_at": &types.AttributeValueMemberS{Value: w.now().UTC().Format(time.RFC3339Nano)},
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("replace random quiz feed: %w", err)
	}
	return Result{QuizCount: len(quizValues)}, nil
}

func publicQuizItem(item map[string]types.AttributeValue) (map[string]types.AttributeValue, error) {
	publicItem := make(map[string]types.AttributeValue, len(publicQuizFields))
	for _, field := range publicQuizFields {
		value, ok := item[field]
		if !ok {
			return nil, fmt.Errorf("quiz item is missing public field %q", field)
		}
		publicItem[field] = value
	}
	return publicItem, nil
}
