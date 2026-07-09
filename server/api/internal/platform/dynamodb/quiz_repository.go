package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

type QuizRepository struct {
	client    *awsdynamodb.Client
	tableName string
}

func NewQuizRepository(client *awsdynamodb.Client, tableName string) *QuizRepository {
	return &QuizRepository{client: client, tableName: tableName}
}

func (r *QuizRepository) Save(ctx context.Context, q quiz.Quiz) error {
	_, err := r.client.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"quiz_id":           stringAttr(q.ID),
			"creator_user_id":   stringAttr(q.CreatorUserID),
			"image_id":          stringAttr(q.ImageID),
			"question":          stringAttr(q.Question),
			"answer":            stringAttr(q.Answer),
			"choices":           choicesAttr(q.Choices),
			"difficulty":        stringAttr(string(q.Difficulty)),
			"crop_x_ratio":      floatAttr(q.Crop.X),
			"crop_y_ratio":      floatAttr(q.Crop.Y),
			"crop_width_ratio":  floatAttr(q.Crop.Width),
			"crop_height_ratio": floatAttr(q.Crop.Height),
			"cropped_image_key": stringAttr(q.CroppedImageKey),
			"status":            stringAttr(string(q.Status)),
			"created_at":        timeAttr(q.CreatedAt),
			"updated_at":        timeAttr(q.UpdatedAt),
		},
	})
	return err
}

func (r *QuizRepository) FindByID(ctx context.Context, id string) (quiz.Quiz, error) {
	out, err := r.client.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"quiz_id": stringAttr(id),
		},
	})
	if err != nil {
		return quiz.Quiz{}, err
	}
	if len(out.Item) == 0 {
		return quiz.Quiz{}, quiz.ErrQuizNotFound
	}
	return quizFromItem(out.Item), nil
}

func (r *QuizRepository) FindByCreatorUserID(ctx context.Context, creatorUserID string) ([]quiz.Quiz, error) {
	paginator := awsdynamodb.NewScanPaginator(r.client, &awsdynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("creator_user_id = :creator_user_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":creator_user_id": stringAttr(creatorUserID),
		},
	})
	quizzes := make([]quiz.Quiz, 0)
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range out.Items {
			quizzes = append(quizzes, quizFromItem(item))
		}
	}
	return quizzes, nil
}

func (r *QuizRepository) Update(ctx context.Context, q quiz.Quiz) error {
	return r.Save(ctx, q)
}

func quizFromItem(item map[string]types.AttributeValue) quiz.Quiz {
	return quiz.Quiz{
		ID:              getString(item, "quiz_id"),
		CreatorUserID:   getString(item, "creator_user_id"),
		ImageID:         getString(item, "image_id"),
		Question:        getString(item, "question"),
		Answer:          getString(item, "answer"),
		Choices:         getChoices(item, "choices"),
		Difficulty:      quiz.Difficulty(getString(item, "difficulty")),
		Crop:            quizCrop(item),
		CroppedImageKey: getString(item, "cropped_image_key"),
		Status:          quiz.Status(getString(item, "status")),
		CreatedAt:       getTime(item, "created_at"),
		UpdatedAt:       getTime(item, "updated_at"),
	}
}
