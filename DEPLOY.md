# デプロイ・アップデート手順

## 初回セットアップ

```bash
git clone git@github.com:mitsuiJao/memos.git
cd memos
./scripts/build-local.sh
docker compose up -d
```

## アップデート

```bash
git pull
./scripts/build-local.sh
docker compose up -d
```

## upstream（本家）の変更を取り込む

```bash
git fetch upstream
git merge upstream/main
./scripts/build-local.sh
docker compose up -d
```
