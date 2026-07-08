# guess-the-celebrity
Now you can guess the celebrity

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
