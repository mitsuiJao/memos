# Docker ビルド詳細

## 概要

本番環境ではビルド済みイメージを GHCR (GitHub Container Registry) から pull して使う。
ビルドはリソースのあるマシン（開発機）で行い、本番サーバーはビルドしない。

```
開発機  → ビルド → ghcr.io/mitsuijao/memos:latest
本番機             ↓ docker compose pull
                   docker compose up -d
```

---

## ファイル構成

| ファイル | 役割 |
|---|---|
| `Dockerfile.standalone` | フロント + バックエンドを含む完結したマルチステージビルド定義 |
| `scripts/build-local.sh` | buildx なし環境向けのビルドスクリプト |
| `docker-compose.yml` | GHCR イメージを使った本番用サービス定義 |

---

## Dockerfile.standalone の構造

```
Stage 1: frontend-builder (node:24-alpine)
  - pnpm@11.0.1 で依存インストール
  - pnpm release → /build/server/router/frontend/dist に出力
    ※ フロントのビルド成果物は Go バイナリに埋め込まれる

Stage 2: backend-builder (golang:1.26.2-alpine)
  - go mod download
  - Stage 1 の dist をコピー
  - CGO_ENABLED=0 でスタティックビルド → バイナリ 1 本に完結

Stage 3: runtime (alpine:3.21)
  - バイナリと entrypoint.sh のみ
  - non-root ユーザー (uid=10001) で実行
  - データは /var/opt/memos に永続化
```

---

## ビルドと push の手順

### 前提
- Docker がインストールされていること（buildx 不要）
- GHCR に `docker login` 済みであること

### GHCR へのログイン（初回のみ）

GitHub で PAT を作成（Settings → Developer settings → Tokens (classic)、`write:packages` 権限）してから：

```bash
echo <PAT> | docker login ghcr.io -u mitsuijao --password-stdin
```

### ビルド

```bash
./scripts/build-local.sh
# → memos-local:latest イメージが作られる
```

内部では以下を実行している：
```bash
DOCKER_BUILDKIT=0 docker build --network=host -f Dockerfile.standalone -t memos-local .
```

`DOCKER_BUILDKIT=0` と `--network=host` はこの開発環境の制約による指定。
buildx があり bridge ネットワークが使える環境では `docker compose build` でも可。

### GHCR に push

```bash
docker tag memos-local ghcr.io/mitsuijao/memos:latest
docker push ghcr.io/mitsuijao/memos:latest
```

---

## ボリュームとデータ

| パス | 内容 |
|---|---|
| `/var/opt/memos` | SQLite DB、アップロードファイルなど全データ |

旧バージョン（`neosmemo/memos:stable`）で `-v /var/opt/memos:/var/opt/memos` を使っていた場合、同じパスをマウントすればデータはそのまま引き継がれる。

---

## 本番サーバーの要件

- Docker がインストールされていること
- `docker-compose` または `docker compose` が使えること
- GHCR パッケージが public の場合: ログイン不要
- GHCR パッケージが private の場合: `docker login ghcr.io` が必要
