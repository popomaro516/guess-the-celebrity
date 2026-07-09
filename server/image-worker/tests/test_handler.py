import json
import logging
from io import BytesIO

import pytest
from PIL import Image

from image_worker import handler
from image_worker.handler import (
    Crop,
    InvalidCropJobError,
    Worker,
    crop_box,
    lambda_handler,
    mask_outside_crop_to_webp,
    parse_crop_job,
)
from lambda_function import lambda_handler as default_lambda_handler


def test_parse_crop_job_accepts_expected_payload():
    job = parse_crop_job(
        json.dumps(
            {
                "quiz_id": "quiz_123",
                "source_image_key": "originals/anonymous/img_123/source.jpg",
                "output_image_key": "quizzes/quiz_123/crop.webp",
                "crop": {"x": 0.25, "y": 0.1, "width": 0.5, "height": 0.4},
            }
        )
    )

    assert job.quiz_id == "quiz_123"
    assert job.crop == Crop(x=0.25, y=0.1, width=0.5, height=0.4)


def test_parse_crop_job_rejects_out_of_bounds_crop():
    with pytest.raises(InvalidCropJobError):
        parse_crop_job(
            json.dumps(
                {
                    "quiz_id": "quiz_123",
                    "source_image_key": "source.jpg",
                    "output_image_key": "crop.webp",
                    "crop": {"x": 0.8, "y": 0.1, "width": 0.3, "height": 0.4},
                }
            )
        )


def test_parse_crop_job_rejects_non_numeric_crop():
    with pytest.raises(InvalidCropJobError):
        parse_crop_job(
            json.dumps(
                {
                    "quiz_id": "quiz_123",
                    "source_image_key": "source.jpg",
                    "output_image_key": "crop.webp",
                    "crop": {"x": "bad", "y": 0, "width": 1, "height": 1},
                }
            )
        )


def test_crop_box_converts_ratios_to_pixels():
    assert crop_box(400, 200, Crop(x=0.25, y=0.1, width=0.5, height=0.4)) == (100, 20, 300, 100)


def test_mask_outside_crop_to_webp_outputs_full_size_webp():
    image = Image.new("RGB", (400, 200), color=(255, 0, 0))
    source = BytesIO()
    image.save(source, format="PNG")

    got = mask_outside_crop_to_webp(source.getvalue(), Crop(x=0.25, y=0.1, width=0.5, height=0.4))

    with Image.open(BytesIO(got)) as masked:
        assert masked.format == "WEBP"
        assert masked.size == (400, 200)


def test_mask_outside_crop_to_webp_blacks_out_everything_except_crop():
    image = Image.new("RGB", (200, 200))
    pixels = image.load()
    for y in range(200):
        for x in range(200):
            if x < 100 and y < 100:
                pixels[x, y] = (255, 0, 0)
            elif x >= 100 and y < 100:
                pixels[x, y] = (0, 255, 0)
            elif x < 100 and y >= 100:
                pixels[x, y] = (0, 0, 255)
            else:
                pixels[x, y] = (255, 255, 0)

    source = BytesIO()
    image.save(source, format="PNG")

    got = mask_outside_crop_to_webp(source.getvalue(), Crop(x=0.5, y=0.5, width=0.5, height=0.5))

    with Image.open(BytesIO(got)) as masked:
        masked = masked.convert("RGB")
        r, g, b = masked.getpixel((150, 150))
        outside_r, outside_g, outside_b = masked.getpixel((50, 50))
        assert masked.size == (200, 200)
        assert r > 200
        assert g > 200
        assert b < 80
        assert outside_r < 20
        assert outside_g < 20
        assert outside_b < 20


def test_worker_uploads_crop_and_marks_ready():
    source = BytesIO()
    Image.new("RGB", (100, 80), color=(0, 255, 0)).save(source, format="PNG")
    s3 = FakeS3({"source.png": source.getvalue()})
    dynamodb = FakeDynamoDB()
    worker = Worker(s3, dynamodb, "bucket", "table")

    worker.handle_message(
        json.dumps(
            {
                "quiz_id": "quiz_123",
                "source_image_key": "source.png",
                "output_image_key": "crop.webp",
                "crop": {"x": 0, "y": 0, "width": 1, "height": 1},
            }
        )
    )

    assert "crop.webp" in s3.objects
    assert s3.put_content_types["crop.webp"] == "image/webp"
    assert dynamodb.status_updates[-1] == ("quiz_123", "ready")


def test_worker_marks_failed_when_crop_processing_fails():
    s3 = FakeS3({"source.png": b"not an image"})
    dynamodb = FakeDynamoDB()
    worker = Worker(s3, dynamodb, "bucket", "table")

    with pytest.raises(Exception):
        worker.handle_message(
            json.dumps(
                {
                    "quiz_id": "quiz_123",
                    "source_image_key": "source.png",
                    "output_image_key": "crop.webp",
                    "crop": {"x": 0, "y": 0, "width": 1, "height": 1},
                }
            )
        )

    assert dynamodb.status_updates[-1] == ("quiz_123", "failed")
    assert "crop.webp" not in s3.objects


def test_lambda_handler_returns_partial_batch_failures(monkeypatch):
    fake_worker = FakeWorker(fail_bodies={"bad"})
    monkeypatch.setattr(handler.Worker, "from_env", lambda: fake_worker)

    got = lambda_handler(
        {
            "Records": [
                {"messageId": "msg-1", "body": "ok"},
                {"messageId": "msg-2", "body": "bad"},
            ]
        },
        None,
    )

    assert got == {"batchItemFailures": [{"itemIdentifier": "msg-2"}]}
    assert fake_worker.handled_messages == [("ok", "msg-1"), ("bad", "msg-2")]


def test_default_lambda_handler_entrypoint_matches_worker_handler():
    assert default_lambda_handler is lambda_handler


def test_json_log_formatter_includes_extra_fields():
    record = logging.LogRecord("image_worker", logging.INFO, "handler.py", 10, "crop job completed", (), None)
    record.quiz_id = "quiz_123"
    record.duration_ms = 42

    got = json.loads(handler.JSONLogFormatter().format(record))

    assert got["level"] == "INFO"
    assert got["msg"] == "crop job completed"
    assert got["logger"] == "image_worker"
    assert got["quiz_id"] == "quiz_123"
    assert got["duration_ms"] == 42


class FakeBody:
    def __init__(self, value):
        self.value = value

    def read(self):
        return self.value


class FakeS3:
    def __init__(self, objects):
        self.objects = dict(objects)
        self.put_content_types = {}

    def get_object(self, Bucket, Key):
        return {"Body": FakeBody(self.objects[Key])}

    def put_object(self, Bucket, Key, Body, ContentType):
        self.objects[Key] = Body
        self.put_content_types[Key] = ContentType


class FakeDynamoDB:
    def __init__(self):
        self.status_updates = []

    def update_item(
        self,
        TableName,
        Key,
        UpdateExpression,
        ExpressionAttributeNames,
        ExpressionAttributeValues,
    ):
        quiz_id = Key["quiz_id"]["S"]
        status = ExpressionAttributeValues[":status"]["S"]
        self.status_updates.append((quiz_id, status))
        return {}


class FakeWorker:
    def __init__(self, fail_bodies):
        self.fail_bodies = set(fail_bodies)
        self.handled_messages = []

    def handle_message(self, body, message_id=""):
        self.handled_messages.append((body, message_id))
        if body in self.fail_bodies:
            raise RuntimeError("boom")
