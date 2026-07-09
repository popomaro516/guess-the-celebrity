from datetime import datetime, timezone

import pytest

from feed_worker import handler
from feed_worker.handler import FeedWorker, lambda_handler
from lambda_function import lambda_handler as default_lambda_handler


def test_worker_queries_latest_ten_published_quizzes_and_replaces_feed():
    dynamodb = FakeDynamoDB(
        [
            quiz_item(
                quiz_id="quiz_2",
                question="二問目",
                choices=["A", "B", "C", "D"],
                difficulty="hard",
                cropped_image_key="quizzes/quiz_2/crop.webp",
            ),
            quiz_item(
                quiz_id="quiz_1",
                question="一問目",
                choices=["E", "F", "G", "H"],
                difficulty="normal",
                cropped_image_key="quizzes/quiz_1/crop.webp",
            ),
        ]
    )
    worker = FeedWorker(
        dynamodb,
        quizzes_table_name="quizzes",
        feed_table_name="feed",
        now=lambda: datetime(2026, 7, 9, 12, 0, tzinfo=timezone.utc),
    )

    result = worker.refresh()

    assert result == {"quiz_count": 2}
    assert dynamodb.query_calls == [
        {
            "TableName": "quizzes",
            "IndexName": "status-created-at-index",
            "KeyConditionExpression": "#status = :published",
            "ExpressionAttributeNames": {"#status": "status"},
            "ExpressionAttributeValues": {":published": {"S": "published"}},
            "ProjectionExpression": "quiz_id, question, choices, difficulty, cropped_image_key",
            "ScanIndexForward": False,
            "Limit": 10,
        }
    ]
    assert dynamodb.put_calls == [
        {
            "TableName": "feed",
            "Item": {
                "feed_id": {"S": "random"},
                "quizzes": {
                    "L": [
                        {
                            "M": {
                                "quiz_id": {"S": "quiz_2"},
                                "question": {"S": "二問目"},
                                "choices": {"L": [{"S": "A"}, {"S": "B"}, {"S": "C"}, {"S": "D"}]},
                                "difficulty": {"S": "hard"},
                                "cropped_image_key": {"S": "quizzes/quiz_2/crop.webp"},
                            }
                        },
                        {
                            "M": {
                                "quiz_id": {"S": "quiz_1"},
                                "question": {"S": "一問目"},
                                "choices": {"L": [{"S": "E"}, {"S": "F"}, {"S": "G"}, {"S": "H"}]},
                                "difficulty": {"S": "normal"},
                                "cropped_image_key": {"S": "quizzes/quiz_1/crop.webp"},
                            }
                        },
                    ]
                },
                "updated_at": {"S": "2026-07-09T12:00:00Z"},
            },
        }
    ]


def test_worker_writes_empty_feed_when_no_published_quizzes_exist():
    dynamodb = FakeDynamoDB([])
    worker = FeedWorker(
        dynamodb,
        quizzes_table_name="quizzes",
        feed_table_name="feed",
        now=lambda: datetime(2026, 7, 9, 12, 0, tzinfo=timezone.utc),
    )

    result = worker.refresh()

    assert result == {"quiz_count": 0}
    assert dynamodb.put_calls[0]["Item"]["quizzes"] == {"L": []}


def test_worker_does_not_replace_feed_when_quiz_is_missing_public_data():
    item = quiz_item(
        quiz_id="quiz_1",
        question="一問目",
        choices=["A", "B", "C", "D"],
        difficulty="normal",
        cropped_image_key="quizzes/quiz_1/crop.webp",
    )
    del item["choices"]
    dynamodb = FakeDynamoDB([item])
    worker = FeedWorker(dynamodb, quizzes_table_name="quizzes", feed_table_name="feed")

    with pytest.raises(ValueError, match="choices"):
        worker.refresh()

    assert dynamodb.put_calls == []


def test_lambda_handler_builds_worker_from_environment_and_refreshes(monkeypatch):
    fake_worker = FakeWorker()
    monkeypatch.setattr(handler.FeedWorker, "from_env", lambda: fake_worker)

    result = lambda_handler({}, None)

    assert result == {"quiz_count": 3}
    assert fake_worker.refresh_calls == 1


def test_worker_requires_table_names():
    with pytest.raises(ValueError, match="DYNAMODB_QUIZZES_TABLE_NAME"):
        FeedWorker(FakeDynamoDB([]), quizzes_table_name="", feed_table_name="feed")
    with pytest.raises(ValueError, match="DYNAMODB_QUIZ_FEED_TABLE_NAME"):
        FeedWorker(FakeDynamoDB([]), quizzes_table_name="quizzes", feed_table_name="")


def test_default_lambda_handler_entrypoint_matches_worker_handler():
    assert default_lambda_handler is lambda_handler


def quiz_item(quiz_id, question, choices, difficulty, cropped_image_key):
    return {
        "quiz_id": {"S": quiz_id},
        "question": {"S": question},
        "choices": {"L": [{"S": choice} for choice in choices]},
        "difficulty": {"S": difficulty},
        "cropped_image_key": {"S": cropped_image_key},
    }


class FakeDynamoDB:
    def __init__(self, items):
        self.items = items
        self.query_calls = []
        self.put_calls = []

    def query(self, **kwargs):
        self.query_calls.append(kwargs)
        return {"Items": self.items}

    def put_item(self, **kwargs):
        self.put_calls.append(kwargs)
        return {}


class FakeWorker:
    def __init__(self):
        self.refresh_calls = 0

    def refresh(self):
        self.refresh_calls += 1
        return {"quiz_count": 3}
