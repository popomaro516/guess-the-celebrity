import logging
import os
from collections.abc import Callable
from datetime import datetime, timezone
from typing import Any

import boto3

FEED_ID = "random"
MAX_QUIZZES = 10
PUBLISHED_STATUS = "published"
STATUS_CREATED_AT_INDEX = "status-created-at-index"
PUBLIC_QUIZ_FIELDS = (
    "quiz_id",
    "question",
    "choices",
    "difficulty",
    "cropped_image_key",
)

logger = logging.getLogger("feed_worker")
logger.setLevel(logging.INFO)


def lambda_handler(_event: dict[str, Any], _context: Any) -> dict[str, int]:
    return FeedWorker.from_env().refresh()


class FeedWorker:
    def __init__(
        self,
        dynamodb_client: Any,
        quizzes_table_name: str,
        feed_table_name: str,
        now: Callable[[], datetime] | None = None,
    ):
        if not quizzes_table_name:
            raise ValueError("DYNAMODB_QUIZZES_TABLE_NAME is required")
        if not feed_table_name:
            raise ValueError("DYNAMODB_QUIZ_FEED_TABLE_NAME is required")

        self.dynamodb = dynamodb_client
        self.quizzes_table_name = quizzes_table_name
        self.feed_table_name = feed_table_name
        self.now = now or (lambda: datetime.now(timezone.utc))

    @classmethod
    def from_env(cls) -> "FeedWorker":
        return cls(
            dynamodb_client=boto3.client("dynamodb"),
            quizzes_table_name=os.environ.get("DYNAMODB_QUIZZES_TABLE_NAME", ""),
            feed_table_name=os.environ.get("DYNAMODB_QUIZ_FEED_TABLE_NAME", ""),
        )

    def refresh(self) -> dict[str, int]:
        response = self.dynamodb.query(
            TableName=self.quizzes_table_name,
            IndexName=STATUS_CREATED_AT_INDEX,
            KeyConditionExpression="#status = :published",
            ExpressionAttributeNames={"#status": "status"},
            ExpressionAttributeValues={":published": {"S": PUBLISHED_STATUS}},
            ProjectionExpression=", ".join(PUBLIC_QUIZ_FIELDS),
            ScanIndexForward=False,
            Limit=MAX_QUIZZES,
        )
        quizzes = [public_quiz_item(item) for item in response.get("Items", [])[:MAX_QUIZZES]]
        updated_at = self.now().astimezone(timezone.utc).isoformat().replace("+00:00", "Z")

        self.dynamodb.put_item(
            TableName=self.feed_table_name,
            Item={
                "feed_id": {"S": FEED_ID},
                "quizzes": {"L": [{"M": item} for item in quizzes]},
                "updated_at": {"S": updated_at},
            },
        )
        logger.info("random quiz feed refreshed", extra={"quiz_count": len(quizzes)})
        return {"quiz_count": len(quizzes)}


def public_quiz_item(item: dict[str, Any]) -> dict[str, Any]:
    missing = [field for field in PUBLIC_QUIZ_FIELDS if field not in item]
    if missing:
        raise ValueError(f"quiz item is missing public fields: {', '.join(missing)}")
    return {field: item[field] for field in PUBLIC_QUIZ_FIELDS}
