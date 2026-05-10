# デプロイ・アップデート手順

## 初回セットアップ（本番サーバー）

```bash
git clone git@github.com:mitsuiJao/memos.git
cd memos
docker compose up -d
```

## アップデート

**このマシンで（ビルド → push）:**
```bash
git pull
./scripts/build-local.sh
docker tag memos-local ghcr.io/mitsuijao/memos:latest
docker push ghcr.io/mitsuijao/memos:latest
```

**本番サーバーで（pull → 再起動）:**
```bash
cd memos
git pull
docker compose pull
docker compose up -d
```

## upstream（本家）の変更を取り込む

```bash
# このマシンで
git fetch upstream
git merge upstream/main
./scripts/build-local.sh
docker tag memos-local ghcr.io/mitsuijao/memos:latest
docker push ghcr.io/mitsuijao/memos:latest

# 本番サーバーで
docker compose pull && docker compose up -d
```
