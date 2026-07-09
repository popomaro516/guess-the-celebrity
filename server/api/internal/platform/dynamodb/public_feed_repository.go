package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

type PublicFeedRepository struct {
	client    *awsdynamodb.Client
	tableName string
}

func NewPublicFeedRepository(
	client *awsdynamodb.Client,
	tableName string,
) *PublicFeedRepository {
	return &PublicFeedRepository{client: client, tableName: tableName}
}

func (r *PublicFeedRepository) FindPublicQuizCandidates(
	ctx context.Context,
	limit int,
) ([]quiz.PublicQuiz, error) {
	if limit <= 0 {
		return []quiz.PublicQuiz{}, nil
	}
	out, err := r.client.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"feed_id": stringAttr(publicFeedID),
		},
	})
	if err != nil {
		return nil, err
	}

	quizzes := getPublicQuizzes(out.Item, "quizzes")
	if len(quizzes) > limit {
		quizzes = quizzes[:limit]
	}
	return quizzes, nil
}

func getPublicQuizzes(item map[string]types.AttributeValue, name string) []quiz.PublicQuiz {
	entries, ok := item[name].(*types.AttributeValueMemberL)
	if !ok {
		return []quiz.PublicQuiz{}
	}
	quizzes := make([]quiz.PublicQuiz, 0, len(entries.Value))
	for _, entry := range entries.Value {
		value, ok := entry.(*types.AttributeValueMemberM)
		if !ok {
			continue
		}
		quizzes = append(quizzes, quiz.PublicQuiz{
			ID:              getString(value.Value, "quiz_id"),
			Question:        getString(value.Value, "question"),
			Choices:         getChoices(value.Value, "choices"),
			Difficulty:      quiz.Difficulty(getString(value.Value, "difficulty")),
			CroppedImageKey: getString(value.Value, "cropped_image_key"),
		})
	}
	return quizzes
}
