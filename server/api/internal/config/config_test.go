package config

import "testing"

func TestDynamoDBConfigRequiresAllTableNames(t *testing.T) {
	cfg := Config{
		DynamoDBImagesTableName:   "guess_the_celebrity_images",
		DynamoDBQuizzesTableName:  "guess_the_celebrity_quizzes",
		DynamoDBQuizFeedTableName: "guess_the_celebrity_quiz_feed",
	}

	if !cfg.HasDynamoDBConfig() {
		t.Fatal("HasDynamoDBConfig() = false, want true")
	}
	if !cfg.HasCompleteDynamoDBConfig() {
		t.Fatal("HasCompleteDynamoDBConfig() = false, want true")
	}

	cfg.DynamoDBQuizFeedTableName = ""
	if !cfg.HasDynamoDBConfig() {
		t.Fatal("partial HasDynamoDBConfig() = false, want true")
	}
	if cfg.HasCompleteDynamoDBConfig() {
		t.Fatal("partial HasCompleteDynamoDBConfig() = true, want false")
	}
}

func TestLoadDynamoDBTableNames(t *testing.T) {
	t.Setenv("DYNAMODB_IMAGES_TABLE_NAME", "images")
	t.Setenv("DYNAMODB_QUIZZES_TABLE_NAME", "quizzes")
	t.Setenv("DYNAMODB_QUIZ_FEED_TABLE_NAME", "quiz-feed")

	cfg := Load()

	if cfg.DynamoDBImagesTableName != "images" {
		t.Fatalf("DynamoDBImagesTableName = %q", cfg.DynamoDBImagesTableName)
	}
	if cfg.DynamoDBQuizzesTableName != "quizzes" {
		t.Fatalf("DynamoDBQuizzesTableName = %q", cfg.DynamoDBQuizzesTableName)
	}
	if cfg.DynamoDBQuizFeedTableName != "quiz-feed" {
		t.Fatalf("DynamoDBQuizFeedTableName = %q", cfg.DynamoDBQuizFeedTableName)
	}
}
