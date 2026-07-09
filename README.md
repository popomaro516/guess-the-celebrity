# guess-the-celebrity
Now you can guess the celebrity

## Cognito authentication

The authoring endpoints require an Amazon Cognito User Pools access token:

```text
POST /uploads/presign
POST /images/{image_id}/complete
POST /quizzes
POST /quizzes/{quiz_id}/publish
```

Configure the API with:

```text
COGNITO_USER_POOL_ID=ap-northeast-1_example
COGNITO_APP_CLIENT_ID=exampleclientid
```

Send the access token as `Authorization: Bearer <token>`. Authentication can only
be bypassed explicitly for local development with `AUTH_DISABLED=true`.
New quizzes store the verified access token's `sub` claim as `creator_user_id`.

## DynamoDB tables

The API uses three dedicated DynamoDB tables:

```text
guess_the_celebrity_images
  partition key: image_id (String)

guess_the_celebrity_quizzes
  partition key: quiz_id (String)
  GSI status-created-at-index:
    partition key: status (String)

guess_the_celebrity_quiz_feed
  partition key: feed_id (String)
```

Configure the API Lambda with:

```text
DYNAMODB_IMAGES_TABLE_NAME=guess_the_celebrity_images
DYNAMODB_QUIZZES_TABLE_NAME=guess_the_celebrity_quizzes
DYNAMODB_QUIZ_FEED_TABLE_NAME=guess_the_celebrity_quiz_feed
```

Configure the image worker Lambda with:

```text
DYNAMODB_QUIZZES_TABLE_NAME=guess_the_celebrity_quizzes
```
