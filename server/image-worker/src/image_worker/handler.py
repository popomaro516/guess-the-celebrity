import json
import logging
import os
from dataclasses import dataclass
from datetime import datetime, timezone
from io import BytesIO
from typing import Any

import boto3
from PIL import Image, ImageOps

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


@dataclass(frozen=True)
class Crop:
    x: float
    y: float
    width: float
    height: float


@dataclass(frozen=True)
class CropJob:
    quiz_id: str
    source_image_key: str
    output_image_key: str
    crop: Crop


class InvalidCropJobError(ValueError):
    pass


def lambda_handler(event: dict[str, Any], _context: Any) -> dict[str, list[dict[str, str]]]:
    worker = Worker.from_env()
    failures: list[dict[str, str]] = []

    for record in event.get("Records", []):
        message_id = record.get("messageId", "")
        try:
            worker.handle_message(record.get("body", ""))
        except Exception:
            logger.exception("failed to process crop job", extra={"message_id": message_id})
            if message_id:
                failures.append({"itemIdentifier": message_id})

    return {"batchItemFailures": failures}


class Worker:
    def __init__(self, s3_client: Any, dynamodb_client: Any, bucket: str, table_name: str):
        if not bucket:
            raise ValueError("S3_BUCKET is required")
        if not table_name:
            raise ValueError("DYNAMODB_TABLE_NAME is required")

        self.s3 = s3_client
        self.dynamodb = dynamodb_client
        self.bucket = bucket
        self.table_name = table_name

    @classmethod
    def from_env(cls) -> "Worker":
        return cls(
            s3_client=boto3.client("s3"),
            dynamodb_client=boto3.client("dynamodb"),
            bucket=os.environ.get("S3_BUCKET", ""),
            table_name=os.environ.get("DYNAMODB_TABLE_NAME", ""),
        )

    def handle_message(self, body: str) -> None:
        job = parse_crop_job(body)
        try:
            image_bytes = self._download(job.source_image_key)
            cropped = crop_to_webp(image_bytes, job.crop)
            self._upload(job.output_image_key, cropped)
            self._mark_quiz_status(job.quiz_id, "ready")
        except Exception:
            self._mark_quiz_status(job.quiz_id, "failed")
            raise

    def _download(self, key: str) -> bytes:
        response = self.s3.get_object(Bucket=self.bucket, Key=key)
        return response["Body"].read()

    def _upload(self, key: str, body: bytes) -> None:
        self.s3.put_object(
            Bucket=self.bucket,
            Key=key,
            Body=body,
            ContentType="image/webp",
        )

    def _mark_quiz_status(self, quiz_id: str, status: str) -> None:
        self.dynamodb.update_item(
            TableName=self.table_name,
            Key={
                "PK": {"S": f"QUIZ#{quiz_id}"},
                "SK": {"S": "METADATA"},
            },
            UpdateExpression="SET #status = :status, updated_at = :updated_at",
            ExpressionAttributeNames={"#status": "status"},
            ExpressionAttributeValues={
                ":status": {"S": status},
                ":updated_at": {"S": now_iso()},
            },
        )


def parse_crop_job(body: str) -> CropJob:
    try:
        payload = json.loads(body)
    except json.JSONDecodeError as exc:
        raise InvalidCropJobError("message body is not valid JSON") from exc

    crop_payload = payload.get("crop") or {}
    try:
        crop = Crop(
            x=float(crop_payload.get("x", -1)),
            y=float(crop_payload.get("y", -1)),
            width=float(crop_payload.get("width", 0)),
            height=float(crop_payload.get("height", 0)),
        )
    except (TypeError, ValueError) as exc:
        raise InvalidCropJobError("crop values must be numeric ratios") from exc

    job = CropJob(
        quiz_id=str(payload.get("quiz_id") or ""),
        source_image_key=str(payload.get("source_image_key") or ""),
        output_image_key=str(payload.get("output_image_key") or ""),
        crop=crop,
    )
    validate_crop_job(job)
    return job


def validate_crop_job(job: CropJob) -> None:
    if not job.quiz_id:
        raise InvalidCropJobError("quiz_id is required")
    if not job.source_image_key:
        raise InvalidCropJobError("source_image_key is required")
    if not job.output_image_key:
        raise InvalidCropJobError("output_image_key is required")

    crop = job.crop
    if crop.x < 0 or crop.y < 0 or crop.width <= 0 or crop.height <= 0:
        raise InvalidCropJobError("crop values must be positive ratios")
    if crop.x + crop.width > 1 or crop.y + crop.height > 1:
        raise InvalidCropJobError("crop must fit within image bounds")


def crop_to_webp(image_bytes: bytes, crop: Crop) -> bytes:
    with Image.open(BytesIO(image_bytes)) as img:
        img = ImageOps.exif_transpose(img)
        width, height = img.size
        box = crop_box(width, height, crop)
        cropped = img.crop(box)
        if cropped.mode not in ("RGB", "RGBA"):
            cropped = cropped.convert("RGB")

        out = BytesIO()
        cropped.save(out, format="WEBP", quality=85, method=6)
        return out.getvalue()


def crop_box(image_width: int, image_height: int, crop: Crop) -> tuple[int, int, int, int]:
    left = round(image_width * crop.x)
    top = round(image_height * crop.y)
    right = round(image_width * (crop.x + crop.width))
    bottom = round(image_height * (crop.y + crop.height))

    left = max(0, min(left, image_width - 1))
    top = max(0, min(top, image_height - 1))
    right = max(left + 1, min(right, image_width))
    bottom = max(top + 1, min(bottom, image_height))
    return left, top, right, bottom


def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")
