# デプロイ・アップデート手順

> Docker ビルドの詳細（Dockerfile の構造・手動ビルド手順など）は [docs/docker-build.md](docs/docker-build.md) を参照。

## 初回セットアップ（本番サーバー）

```bash
git clone git@github.com:mitsuiJao/memos.git
cd memos
docker compose pull
docker compose up -d
```

## アップデート

**このマシンで（コードを push → CI/CD が自動ビルド）:**
```bash
git pull
git push origin main
# → GitHub Actions (ci.yml) が lint/test → Docker build → ghcr.io/mitsuijao/memos:latest push を自動実行
```

**本番サーバーで（CI/CD 完了後）:**
```bash
cd memos
docker compose pull
docker compose up -d
```

## upstream（本家）の変更を取り込む

```bash
# このマシンで
git fetch upstream
git merge upstream/main
git push origin main
# → CI/CD が自動ビルド・push

# 本番サーバーで（CI/CD 完了後）
docker compose pull && docker compose up -d
```

## 手動ビルド（CI/CD が使えない場合）

```bash
# このマシンで
./scripts/build-local.sh
docker tag memos-local ghcr.io/mitsuijao/memos:latest
docker push ghcr.io/mitsuijao/memos:latest

# 本番サーバーで
docker compose pull && docker compose up -d
```
