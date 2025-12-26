# パブリックリポジトリ化の準備チェックリスト

## ✅ 完了済み項目

### セキュリティ
- [x] センシティブ情報の確認完了（APIキー、パスワード等なし）
- [x] 使用IPアドレスは全て例示用の標準的なもの
- [x] MITライセンス設定済み
- [x] SECURITY.mdファイル作成済み
- [x] .gitignoreファイル強化済み

### ドキュメント
- [x] 包括的なREADME.md
- [x] CONTRIBUTING.md（貢献ガイドライン）
- [x] CHANGELOG.md（変更履歴）
- [x] Issue/PRテンプレート

### CI/CD
- [x] GitHub Actions設定済み
- [x] 自動テスト・ビルド・リリース
- [x] マルチプラットフォーム対応

## 🔧 推奨設定（パブリック化前）

### 1. リポジトリ設定の更新

```bash
# リポジトリ説明の追加
gh repo edit --description "A Go implementation of deadman ping monitoring tool with terminal UI and Prometheus metrics support"

# ホームページURLの設定（リリース後）
# gh repo edit --homepage "https://github.com/doridoridoriand/deadman-go"

# トピックの追加
gh repo edit --add-topic "go,monitoring,ping,terminal-ui,prometheus,networking,deadman"
```

### 2. ブランチ保護の設定（mainブランチ）

```bash
# mainブランチの保護設定
gh api repos/doridoridoriand/deadman-go/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["test"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1,"dismiss_stale_reviews":true}' \
  --field restrictions=null
```

### 3. リポジトリ機能の最適化

```bash
# Wikiを無効化（READMEで十分）
gh repo edit --enable-wiki=false

# Projectsを無効化（Issuesで管理）
gh repo edit --enable-projects=false

# Discussionsを有効化（コミュニティ用）
gh repo edit --enable-discussions=true
```

### 4. 初回リリースの作成

```bash
# initial-developブランチから最新の変更をmainにマージ
git checkout main
git merge initial-develop
git push origin main

# 初回リリースの作成
./scripts/release.sh v0.0.1
```

## ⚠️ 注意事項

### デフォルトブランチについて
- 現在のデフォルトブランチ: `main`
- 開発ブランチ: `initial-develop`
- パブリック化前に`initial-develop`の内容を`main`にマージすることを推奨

### ブランチ整理
以下の不要なブランチを削除することを推奨：
- `copilot/sub-pr-2`
- `copilot/sub-pr-2-again`

```bash
# 不要ブランチの削除
gh api repos/doridoridoriand/deadman-go/git/refs/heads/copilot/sub-pr-2 --method DELETE
gh api repos/doridoridoriand/deadman-go/git/refs/heads/copilot/sub-pr-2-again --method DELETE
```

## 🚀 パブリック化手順

### 1. 最終確認
- [ ] 全てのテストが通ることを確認
- [ ] ドキュメントの最終レビュー
- [ ] 不要ブランチの削除

### 2. パブリック化実行
```bash
gh repo edit --visibility public
```

### 3. パブリック化後の作業
- [ ] 初回リリース（v0.0.1）の作成
- [ ] README.mdのリンク確認
- [ ] GitHub Pagesの設定（必要に応じて）
- [ ] コミュニティ標準の確認

## 📊 現在のリポジトリ状態

- **可視性**: Private
- **Issues**: 有効
- **Wiki**: 有効（無効化推奨）
- **Projects**: 有効（無効化推奨）
- **セキュリティポリシー**: 有効
- **ライセンス**: MIT
- **ブランチ保護**: 未設定（設定推奨）

## 🎯 推奨タイムライン

1. **今すぐ**: .gitignore更新、リポジトリ設定の最適化
2. **パブリック化直前**: ブランチ整理、最終テスト
3. **パブリック化直後**: 初回リリース作成
4. **1週間以内**: コミュニティからのフィードバック対応

---

**注意**: パブリック化は取り消しできません。全ての準備が完了してから実行してください。