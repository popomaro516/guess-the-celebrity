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

func TestLoadCognitoConfig(t *testing.T) {
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("COGNITO_USER_POOL_ID", "us-west-2_example")
	t.Setenv("COGNITO_APP_CLIENT_ID", "client-id")
	t.Setenv("AUTH_DISABLED", "false")

	cfg := Load()

	if !cfg.HasCompleteCognitoConfig() {
		t.Fatal("HasCompleteCognitoConfig() = false, want true")
	}
	if got, want := cfg.CognitoIssuer(), "https://cognito-idp.us-west-2.amazonaws.com/us-west-2_example"; got != want {
		t.Fatalf("CognitoIssuer() = %q, want %q", got, want)
	}
	if cfg.AuthDisabled {
		t.Fatal("AuthDisabled = true, want false")
	}
}

func TestAuthCanBeExplicitlyDisabled(t *testing.T) {
	t.Setenv("AUTH_DISABLED", "true")

	if cfg := Load(); !cfg.AuthDisabled {
		t.Fatal("AuthDisabled = false, want true")
	}
}
