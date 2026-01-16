# タスク: 設定リロードエラー処理のプロパティテスト（プロパティ15）

## タスク概要
- **プロパティ15**: 設定リロードエラー処理
- **検証対象**: 要件 5.4
- **説明**: 無効な設定ファイルでのリロード試行に対して、システムは既存の設定を維持し、適切なエラーログを出力する

## 関連ファイル
- `main.go` - リロード処理の実装（92-101行目、112-114行目）
- `internal/config/parser.go` - 設定ファイルパーサー
- `internal/state/store.go` - 状態管理（UpdateTargets, UpdateTimeout）
- `internal/scheduler/scheduler.go` - スケジューラー（UpdateConfig）

## 実装の詳細

### 現在のリロード処理（main.go）
```go
reload := func() error {
    newCfg, err := parser.LoadConfig(configPath, overrides)
    if err != nil {
        return err  // エラーが返されるが、既存設定は維持される
    }
    sched.UpdateConfig(newCfg.Global, newCfg.Targets)
    store.UpdateTargets(newCfg.Targets)
    store.UpdateTimeout(newCfg.Global.Timeout)
    return nil
}

// エラーハンドリング（112-114行目）
if err := reload(); err != nil {
    fmt.Fprintf(os.Stderr, "failed to reload config: %v\n", err)
}
```

### 要件 5.4
- WHEN 設定リロード時にエラーが発生する THEN システムは既存の設定を維持してエラーログを出力する

## テスト実装方針

### プロパティテストの要件
- **使用ライブラリ**: `gopter`
- **最低反復回数**: 100回
- **タグ形式**: `**Feature: surveiller, Property 15: 設定リロードエラー処理**`

### テストケース
1. **無効な設定ファイルでのリロード**
   - 構文エラーを含む設定ファイル
   - 存在しないファイルパス
   - 権限不足のファイル

2. **既存設定の維持**
   - リロードエラー後も既存のターゲットが監視され続ける
   - 既存の設定値（interval, timeout等）が維持される
   - 既存の状態（RTT履歴等）が維持される

3. **エラーログの出力**
   - エラーメッセージがstderrに出力される
   - エラー内容が適切に記録される

## 参考実装
- `internal/state/store_prop_test.go` - プロパティテストの実装例
- `internal/ui/ui_prop_test.go` - プロパティテストの実装例
- `main_test.go` - mainパッケージのテスト例

## 実装場所
- ファイル: `main_test.go` または `main_prop_test.go`
- 関数名: `TestPropertyConfigReloadErrorHandling`

## 注意事項
- プロパティテストは実行時間が長くなる可能性がある
- 実際のファイルシステム操作が必要な場合は、`t.TempDir()`を使用
- エラーハンドリングの検証には、stderrのキャプチャが必要な場合がある
