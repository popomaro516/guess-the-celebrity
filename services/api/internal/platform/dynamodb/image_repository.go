package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/image"
)

type ImageRepository struct {
	client    *awsdynamodb.Client
	tableName string
}

func NewImageRepository(client *awsdynamodb.Client, tableName string) *ImageRepository {
	return &ImageRepository{client: client, tableName: tableName}
}

func (r *ImageRepository) Save(ctx context.Context, img image.Image) error {
	_, err := r.client.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"PK":                 stringAttr(imagePK(img.ID)),
			"SK":                 stringAttr(metadataSK),
			"type":               stringAttr(imageType),
			"image_id":           stringAttr(img.ID),
			"owner_user_id":      stringAttr(img.OwnerUserID),
			"original_image_key": stringAttr(img.OriginalImageKey),
			"content_type":       stringAttr(img.ContentType),
			"size":               int64Attr(img.Size),
			"status":             stringAttr(string(img.Status)),
			"created_at":         timeAttr(img.CreatedAt),
			"updated_at":         timeAttr(img.UpdatedAt),
		},
	})
	return err
}

func (r *ImageRepository) FindByID(ctx context.Context, id string) (image.Image, error) {
	out, err := r.client.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": stringAttr(imagePK(id)),
			"SK": stringAttr(metadataSK),
		},
	})
	if err != nil {
		return image.Image{}, err
	}
	if len(out.Item) == 0 || getString(out.Item, "type") != imageType {
		return image.Image{}, image.ErrImageNotFound
	}
	return image.Image{
		ID:               getString(out.Item, "image_id"),
		OwnerUserID:      getString(out.Item, "owner_user_id"),
		OriginalImageKey: getString(out.Item, "original_image_key"),
		ContentType:      getString(out.Item, "content_type"),
		Size:             getInt64(out.Item, "size"),
		Status:           image.Status(getString(out.Item, "status")),
		CreatedAt:        getTime(out.Item, "created_at"),
		UpdatedAt:        getTime(out.Item, "updated_at"),
	}, nil
}

func (r *ImageRepository) Update(ctx context.Context, img image.Image) error {
	return r.Save(ctx, img)
}
