# Guess the Celebrity

## 1. 概要

Guess the Celebrityは、画像の一部を切り抜いた画像から、元画像に写っている対象を当てる4択クイズアプリケーションである。

ユーザーは画像をアップロードし、出題範囲、問題文、正解、選択肢、難易度を指定してクイズを作成・公開できる。回答者は公開済みクイズに回答し、正誤結果を確認できる。

## 2. ユーザー

### 2.1 作問者

- ログインする
- 画像をアップロードする
- 出題するcrop範囲を指定する
- 問題文、正解、4つの選択肢、難易度を設定する
- 自分が作成したクイズを公開する

### 2.2 回答者

- 公開済みクイズを取得する
- 4つの選択肢から回答する
- 正誤結果を確認する
- 正解時に正解と元画像を確認する

## 3. 認証・認可

### 3.1 認証

作問操作にはCognitoのアクセストークンを必要とする。

対象APIには次のHTTPヘッダーを付与する。

```http
Authorization: Bearer <access_token>
```

APIはトークンの署名、有効期限、issuer、`token_use`、App Client IDを検証する。認証済みユーザーの識別子にはアクセストークンの`sub`を使用する。

### 3.2 認可

- クイズ作成時、認証済みユーザーの`sub`を`creator_user_id`として保存する
- クイズを公開できるのは、`creator_user_id`が認証済みユーザーの`sub`と一致する場合のみとする
- 所有者以外による公開要求には`403 Forbidden`を返す

## 4. 画像アップロード

画像はpresigned URLを使用してアップロードする。

処理順序は次のとおりとする。

1. クライアントがアップロード対象のファイル情報をAPIへ送信する
2. APIが画像メタデータを`pending_upload`で保存し、presigned PUT URLを返す
3. クライアントがpresigned URLへ画像をPUTする
4. クライアントがアップロード完了をAPIへ通知する
5. APIがS3上のオブジェクト存在を確認する
6. APIが画像ステータスを`uploaded`へ更新する

### 4.1 入力制約

- 対応Content-Type: `image/jpeg`, `image/png`, `image/webp`
- ファイルサイズ: 1 byte以上、10 MiB以下
- presigned URL有効期限: 300秒

## 5. crop指定

crop範囲は元画像の幅と高さに対する比率で指定する。

```json
{
  "x": 0.24,
  "y": 0.18,
  "width": 0.32,
  "height": 0.28
}
```

各値は次の条件を満たす必要がある。

```text
x >= 0
y >= 0
width > 0
height > 0
x + width <= 1
y + height <= 1
```

## 6. クイズ

### 6.1 作成

クイズ作成には、アップロード済みの画像を指定する。

作成時に次の処理を行う。

1. クイズを`processing`で保存する
2. cropジョブをキューへ送信する
3. `quiz_id`とステータスを返す

選択肢は空文字と重複を含まない4件とし、そのうち1件は正解と一致する必要がある。

### 6.2 画像処理

cropジョブは次の順序で処理する。

1. 元画像を取得する
2. crop比率をピクセル座標へ変換する
3. 指定範囲を切り抜く
4. WebP形式で保存する
5. クイズを`ready`へ更新する

処理に失敗した場合、クイズを`failed`へ更新する。

### 6.3 公開

- `ready`のクイズのみ公開できる
- 作成者本人のみ公開できる
- 公開に成功したクイズは`published`へ更新する
- 回答者が取得・回答できるのは`published`のクイズのみとする

### 6.4 ステータス

```text
processing -> ready -> published
     |
     +-----> failed
```

| ステータス | 意味 |
| --- | --- |
| `processing` | crop画像を生成中 |
| `ready` | crop画像の生成が完了し、公開可能 |
| `published` | 公開済み |
| `failed` | crop画像の生成に失敗 |

## 7. ランダム出題

公開済みクイズの候補はfeed workerが最大10件のIDリストとして事前生成する。

APIは次の処理を行う。

1. `feed_id = random`のfeedを取得する
2. `quiz_ids`からランダムに1件選ぶ
3. 対象クイズを取得する
4. 対象が`published`であることを確認する
5. 回答に必要な情報だけを返す

候補が存在しない場合は`404 Not Found`を返す。

## 8. 回答

- 回答APIは認証を必要としない
- 回答対象は`published`のクイズに限る
- 不正解時は`correct: false`のみを返す
- 正解時は正解と元画像URLを返す
- 回答履歴は保存しない

## 9. API

### 9.1 共通

リクエストとレスポンスのContent-Typeは`application/json`とする。ただし、presigned URLを使ったS3へのPUTを除く。

主なエラーステータスは次のとおりとする。

| ステータス | 条件 |
| --- | --- |
| `400 Bad Request` | JSON、入力値、状態遷移が不正 |
| `401 Unauthorized` | アクセストークンがない、または無効 |
| `403 Forbidden` | 認証済みだが操作対象の所有者ではない |
| `404 Not Found` | 対象リソースが存在しない |
| `500 Internal Server Error` | 予期しない内部エラー |

### 9.2 ヘルスチェック

#### `GET /health`

#### `GET /healthz`

Response: `200 OK`

```json
{
  "status": "ok"
}
```

### 9.3 アップロードURL発行

#### `POST /uploads/presign`

認証: 必須

Request:

```json
{
  "filename": "sample.jpg",
  "content_type": "image/jpeg",
  "size": 2048000
}
```

Response: `200 OK`

```json
{
  "image_id": "img_123",
  "upload_url": "https://...",
  "object_key": "originals/anonymous/img_123/source.jpg",
  "expires_in": 300
}
```

### 9.4 アップロード完了

#### `POST /images/{image_id}/complete`

認証: 必須

Response: `200 OK`

```json
{
  "image_id": "img_123",
  "status": "uploaded"
}
```

### 9.5 クイズ作成

#### `POST /quizzes`

認証: 必須

Request:

```json
{
  "image_id": "img_123",
  "question": "この画像に写っているものは何？",
  "answer": "subject_a",
  "choices": ["subject_a", "subject_b", "subject_c", "subject_d"],
  "difficulty": "normal",
  "crop": {
    "x": 0.24,
    "y": 0.18,
    "width": 0.32,
    "height": 0.28
  }
}
```

Response: `201 Created`

```json
{
  "quiz_id": "quiz_123",
  "status": "processing"
}
```

### 9.6 クイズ公開

#### `POST /quizzes/{quiz_id}/publish`

認証: 必須

Response: `200 OK`

```json
{
  "quiz_id": "quiz_123",
  "status": "published"
}
```

### 9.7 ランダムクイズ取得

#### `GET /quizzes/random`

認証: 不要

Response: `200 OK`

```json
{
  "quiz_id": "quiz_123",
  "question": "この画像に写っているものは何？",
  "cropped_image_url": "https://...",
  "choices": ["subject_a", "subject_b", "subject_c", "subject_d"],
  "difficulty": "normal"
}
```

### 9.8 クイズ回答

#### `POST /quizzes/{quiz_id}/answer`

認証: 不要

Request:

```json
{
  "answer": "subject_a"
}
```

正解時のResponse: `200 OK`

```json
{
  "correct": true,
  "correct_answer": "subject_a",
  "original_image_url": "https://..."
}
```

不正解時のResponse: `200 OK`

```json
{
  "correct": false
}
```

## 10. データ

### 10.1 Imagesテーブル

Partition key: `image_id`

| 属性 | 型 | 内容 |
| --- | --- | --- |
| `image_id` | String | 画像ID |
| `owner_user_id` | String | 現在は固定値`anonymous` |
| `original_image_key` | String | 元画像のS3オブジェクトキー |
| `content_type` | String | Content-Type |
| `size` | Number | ファイルサイズ |
| `status` | String | `pending_upload`または`uploaded` |
| `created_at` | String | 作成日時 |
| `updated_at` | String | 更新日時 |

### 10.2 Quizzesテーブル

Partition key: `quiz_id`

| 属性 | 型 | 内容 |
| --- | --- | --- |
| `quiz_id` | String | クイズID |
| `creator_user_id` | String | Cognitoアクセストークンの`sub` |
| `image_id` | String | 元画像ID |
| `question` | String | 問題文 |
| `answer` | String | 正解 |
| `choices` | List | 4つの選択肢 |
| `difficulty` | String | `easy`, `normal`, `hard` |
| `crop_x_ratio` | Number | crop開始位置X |
| `crop_y_ratio` | Number | crop開始位置Y |
| `crop_width_ratio` | Number | crop幅 |
| `crop_height_ratio` | Number | crop高さ |
| `cropped_image_key` | String | crop画像のS3オブジェクトキー |
| `status` | String | クイズステータス |
| `created_at` | String | 作成日時 |
| `updated_at` | String | 更新日時 |

feed workerは`status-created-at-index`を使用して公開済みクイズを取得する。

### 10.3 Quiz Feedテーブル

Partition key: `feed_id`

```json
{
  "feed_id": "random",
  "quiz_ids": ["quiz_123", "quiz_456"],
  "updated_at": "2026-07-08T00:00:00Z"
}
```

## 11. オブジェクトキー

元画像:

```text
originals/anonymous/{image_id}/source.{extension}
```

crop画像:

```text
quizzes/{quiz_id}/crop.webp
```

## 12. cropジョブ

キューへ送信するメッセージ形式は次のとおりとする。

```json
{
  "quiz_id": "quiz_123",
  "source_image_key": "originals/anonymous/img_123/source.jpg",
  "output_image_key": "quizzes/quiz_123/crop.webp",
  "crop": {
    "x": 0.24,
    "y": 0.18,
    "width": 0.32,
    "height": 0.28
  }
}
```
