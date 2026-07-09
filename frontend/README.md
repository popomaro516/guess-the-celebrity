# Frontend

Astroの静的出力とReact Islandsで構成するフロントエンドです。

## Local development

プロジェクトルートから起動します。

```sh
cp frontend/.env.example frontend/.env.local
docker compose up --build frontend api
```

画面は `http://localhost:4321`、APIはフロントの `/api/*` から同一オリジンで
プロキシされます。

## Environment variables

| Name | Purpose |
| --- | --- |
| `PUBLIC_AWS_REGION` | Cognito User PoolのAWSリージョン |
| `PUBLIC_COGNITO_USER_POOL_ID` | Cognito User Pool ID |
| `PUBLIC_COGNITO_USER_POOL_CLIENT_ID` | ブラウザ用App Client ID（secretなし） |
| `PUBLIC_API_BASE_URL` | ブラウザから呼ぶAPIのパス。既定値は`/api` |
| `API_PROXY_TARGET` | ローカル開発サーバーのAPI転送先 |

## Amplify Hosting

リポジトリルートの `amplify.yml` を使用します。Amplify側で上記の
`PUBLIC_*` 変数を設定し、`/api/<*>` から本番APIのHTTPS URLへの200 rewriteを
設定してください。
