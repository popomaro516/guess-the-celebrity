package feed

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestRefreshQueriesLatestPublishedQuizzesAndReplacesFeed(t *testing.T) {
	client := &fakeDynamoDB{
		queryOutput: &dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				quizItem("quiz_2", "二問目", []string{"A", "B", "C", "D"}, "hard", "quizzes/quiz_2/crop.webp"),
				quizItem("quiz_1", "一問目", []string{"E", "F", "G", "H"}, "normal", "quizzes/quiz_1/crop.webp"),
			},
		},
	}
	worker, err := New(client, Config{
		QuizzesTableName: "quizzes",
		FeedTableName:    "feed",
	}, func() time.Time {
		return time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := worker.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	if result.QuizCount != 2 {
		t.Fatalf("QuizCount = %d, want 2", result.QuizCount)
	}
	if len(client.queryInputs) != 1 {
		t.Fatalf("Query calls = %d, want 1", len(client.queryInputs))
	}
	query := client.queryInputs[0]
	if aws.ToString(query.TableName) != "quizzes" ||
		aws.ToString(query.IndexName) != statusCreatedAtIndex ||
		aws.ToString(query.KeyConditionExpression) != "#status = :published" ||
		aws.ToInt32(query.Limit) != 10 ||
		query.ScanIndexForward == nil ||
		*query.ScanIndexForward {
		t.Fatalf("unexpected QueryInput: %+v", query)
	}
	if len(client.putInputs) != 1 {
		t.Fatalf("PutItem calls = %d, want 1", len(client.putInputs))
	}
	put := client.putInputs[0]
	if aws.ToString(put.TableName) != "feed" {
		t.Fatalf("PutItem table = %q, want feed", aws.ToString(put.TableName))
	}
	if got := stringValue(put.Item, "feed_id"); got != "random" {
		t.Fatalf("feed_id = %q, want random", got)
	}
	if got := stringValue(put.Item, "updated_at"); got != "2026-07-09T12:00:00Z" {
		t.Fatalf("updated_at = %q", got)
	}
	quizzes := listValue(put.Item, "quizzes")
	if len(quizzes) != 2 {
		t.Fatalf("len(quizzes) = %d, want 2", len(quizzes))
	}
	first := quizzes[0].(*types.AttributeValueMemberM).Value
	if stringValue(first, "quiz_id") != "quiz_2" || stringValue(first, "question") != "二問目" {
		t.Fatalf("unexpected first quiz: %+v", first)
	}
	if _, exists := first["answer"]; exists {
		t.Fatal("public feed must not contain answer")
	}
}

func TestRefreshWritesEmptyFeed(t *testing.T) {
	client := &fakeDynamoDB{queryOutput: &dynamodb.QueryOutput{}}
	worker, err := New(client, Config{QuizzesTableName: "quizzes", FeedTableName: "feed"}, time.Now)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := worker.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	if result.QuizCount != 0 {
		t.Fatalf("QuizCount = %d, want 0", result.QuizCount)
	}
	if got := listValue(client.putInputs[0].Item, "quizzes"); len(got) != 0 {
		t.Fatalf("len(quizzes) = %d, want 0", len(got))
	}
}

func TestRefreshLogsStructuredSummary(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	client := &fakeDynamoDB{
		queryOutput: &dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				quizItem("quiz_2", "二問目", []string{"A", "B", "C", "D"}, "hard", "quizzes/quiz_2/crop.webp"),
			},
		},
	}
	worker, err := New(client, Config{
		QuizzesTableName: "quizzes",
		FeedTableName:    "feed",
		Logger:           logger,
	}, func() time.Time {
		return time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = worker.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	logs := logOutput.String()
	if !strings.Contains(logs, `"msg":"feed refresh completed"`) {
		t.Fatalf("completion log missing: %s", logs)
	}
	if !strings.Contains(logs, `"quiz_count":1`) {
		t.Fatalf("quiz_count missing from logs: %s", logs)
	}
	if !strings.Contains(logs, `"feed_id":"random"`) {
		t.Fatalf("feed_id missing from logs: %s", logs)
	}
}

func TestRefreshDoesNotReplaceFeedWhenQuizIsMissingPublicData(t *testing.T) {
	item := quizItem("quiz_1", "一問目", []string{"A", "B", "C", "D"}, "normal", "quizzes/quiz_1/crop.webp")
	delete(item, "choices")
	client := &fakeDynamoDB{queryOutput: &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item}}}
	worker, err := New(client, Config{QuizzesTableName: "quizzes", FeedTableName: "feed"}, time.Now)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = worker.Refresh(context.Background())

	if err == nil {
		t.Fatal("Refresh returned nil error")
	}
	if len(client.putInputs) != 0 {
		t.Fatalf("PutItem calls = %d, want 0", len(client.putInputs))
	}
}

func TestNewRequiresTableNames(t *testing.T) {
	tests := []Config{
		{FeedTableName: "feed"},
		{QuizzesTableName: "quizzes"},
	}
	for _, config := range tests {
		if _, err := New(&fakeDynamoDB{}, config, time.Now); err == nil {
			t.Fatalf("New(%+v) returned nil error", config)
		}
	}
}

func quizItem(id, question string, choices []string, difficulty, croppedImageKey string) map[string]types.AttributeValue {
	choiceValues := make([]types.AttributeValue, 0, len(choices))
	for _, choice := range choices {
		choiceValues = append(choiceValues, &types.AttributeValueMemberS{Value: choice})
	}
	return map[string]types.AttributeValue{
		"quiz_id":           &types.AttributeValueMemberS{Value: id},
		"question":          &types.AttributeValueMemberS{Value: question},
		"choices":           &types.AttributeValueMemberL{Value: choiceValues},
		"difficulty":        &types.AttributeValueMemberS{Value: difficulty},
		"cropped_image_key": &types.AttributeValueMemberS{Value: croppedImageKey},
	}
}

func stringValue(item map[string]types.AttributeValue, name string) string {
	value, _ := item[name].(*types.AttributeValueMemberS)
	if value == nil {
		return ""
	}
	return value.Value
}

func listValue(item map[string]types.AttributeValue, name string) []types.AttributeValue {
	value, _ := item[name].(*types.AttributeValueMemberL)
	if value == nil {
		return nil
	}
	return value.Value
}

type fakeDynamoDB struct {
	queryOutput *dynamodb.QueryOutput
	queryErr    error
	putErr      error
	queryInputs []*dynamodb.QueryInput
	putInputs   []*dynamodb.PutItemInput
}

func (f *fakeDynamoDB) Query(
	_ context.Context,
	input *dynamodb.QueryInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.QueryOutput, error) {
	f.queryInputs = append(f.queryInputs, input)
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	if f.queryOutput == nil {
		return nil, errors.New("query output not configured")
	}
	return f.queryOutput, nil
}

func (f *fakeDynamoDB) PutItem(
	_ context.Context,
	input *dynamodb.PutItemInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.PutItemOutput, error) {
	f.putInputs = append(f.putInputs, input)
	return &dynamodb.PutItemOutput{}, f.putErr
}
