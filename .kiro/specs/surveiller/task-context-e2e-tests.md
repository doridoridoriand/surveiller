# タスク: エンドツーエンド統合テスト実装（7.1, 7.2, 7.3）

## タスク概要
エンドツーエンド統合テストの実装。コンポーネント間の連携動作を検証する。

## サブタスク

### 7.1 設定ファイルから監視開始までの統合テスト
- **検証対象**: 要件 1.1, 2.1, 4.1
- **説明**: 設定ファイル読み込み → スケジューラー起動 → ping 実行の流れ

### 7.2 TUI とバックエンドの統合テスト
- **検証対象**: 要件 3.1, 3.4, 3.5
- **説明**: 状態更新 → TUI 表示更新の流れ、キーボード入力処理

### 7.3 設定リロードの統合テスト
- **検証対象**: 要件 5.1, 5.2, 5.3
- **説明**: SIGHUP → 設定再読み込み → 監視更新の流れ

## 関連ファイル

### 主要コンポーネント
- `main.go` - アプリケーションエントリーポイント
- `internal/config/parser.go` - 設定ファイルパーサー
- `internal/scheduler/scheduler.go` - スケジューラー
- `internal/ping/` - Ping実装（ICMP, External, Fallback）
- `internal/state/store.go` - 状態管理
- `internal/ui/ui.go` - TUI実装

### テストファイル
- `main_test.go` - mainパッケージのテスト（既存）
- `internal/config/parser_test.go` - 設定パーサーのテスト
- `internal/scheduler/scheduler_test.go` - スケジューラーのテスト
- `internal/ui/ui_test.go` - UIのテスト

## 実装の詳細

### 7.1 設定ファイルから監視開始までの統合テスト

#### テストフロー
1. 一時的な設定ファイルを作成
2. 設定ファイルを読み込む
3. スケジューラーを起動
4. pingが実行されることを確認
5. 状態が更新されることを確認

#### 必要なモック
- Ping実装のモック（実際のネットワーク通信を避ける）
- タイムアウトの制御

#### 実装例
```go
func TestE2E_ConfigToMonitoring(t *testing.T) {
    // 1. 設定ファイル作成
    configPath := createTempConfig(t, "...")
    
    // 2. コンポーネント初期化
    parser := config.SurveillerParser{}
    cfg, err := parser.LoadConfig(configPath, config.CLIOverrides{})
    // ...
    
    // 3. モックpinger作成
    mockPinger := &MockPinger{}
    
    // 4. スケジューラー起動
    store := state.NewStore(cfg.Targets, cfg.Global.Timeout)
    sched := scheduler.NewScheduler(cfg.Global, cfg.Targets, mockPinger, store)
    
    // 5. 監視開始と検証
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    go sched.Run(ctx)
    
    // pingが実行されることを確認
    // 状態が更新されることを確認
}
```

### 7.2 TUI とバックエンドの統合テスト

#### テストフロー
1. 状態管理ストアを作成
2. TUIを初期化
3. 状態を更新
4. TUIの表示が更新されることを確認
5. キーボード入力をシミュレート

#### 注意事項
- TUIのテストは実際の端末出力を避ける
- tcellのテスト機能を活用
- モックスクリーンを使用

#### 実装例
```go
func TestE2E_TUIAndBackend(t *testing.T) {
    // 1. 状態管理ストア作成
    store := state.NewStore([]config.TargetConfig{...}, timeout)
    
    // 2. TUI初期化
    ui := ui.New(cfg, store, reloadCh)
    
    // 3. 状態更新
    store.UpdateResult("target1", ping.Result{Success: true, RTT: 10*time.Millisecond})
    
    // 4. TUI表示の検証
    snapshot := store.GetSnapshot()
    // TUIの表示内容を検証
    
    // 5. キーボード入力のシミュレート
    // 'q'キーで終了
    // 'r'キーでリロード
}
```

### 7.3 設定リロードの統合テスト

#### テストフロー
1. 初期設定ファイルでアプリケーションを起動
2. 監視を開始
3. 設定ファイルを変更
4. SIGHUPシグナルを送信（またはリロードチャネルに送信）
5. 設定が再読み込みされることを確認
6. 監視が更新されることを確認

#### 注意事項
- シグナルの送信はテスト環境で制限される可能性がある
- リロードチャネルを直接使用する方法も検討

#### 実装例
```go
func TestE2E_ConfigReload(t *testing.T) {
    // 1. 初期設定ファイル作成
    configPath := createTempConfig(t, "initial config...")
    
    // 2. アプリケーション初期化
    // ...
    
    // 3. 監視開始
    // ...
    
    // 4. 設定ファイルを変更
    updateConfigFile(t, configPath, "updated config...")
    
    // 5. リロードをトリガー
    reloadCh <- struct{}{}
    
    // 6. 設定が更新されることを確認
    // 新しいターゲットが追加される
    // 削除されたターゲットの監視が停止される
}
```

## テスト実装ガイドライン

### 統合テストの要件
- **テスト環境**: 一時ディレクトリとモックサーバー使用
- **テスト範囲**: コンポーネント間の連携動作
- **実行時間**: 各テスト5秒以内

### モックの使用
- 外部依存（ネットワーク、ファイルシステム）のみモックを使用
- 実際のネットワーク通信は避ける
- ファイルシステム操作は`t.TempDir()`を使用

### テストヘルパー関数
- `createTempConfig(t *testing.T, content string) string` - 一時設定ファイル作成
- `MockPinger` - Ping実装のモック
- `waitForCondition` - 条件が満たされるまで待機

## 実装場所
- ファイル: `main_e2e_test.go` または `main_test.go`に追加
- テスト関数名:
  - `TestE2E_ConfigToMonitoring`
  - `TestE2E_TUIAndBackend`
  - `TestE2E_ConfigReload`

## 参考実装
- `main_test.go` - mainパッケージのテスト例
- `internal/config/parser_test.go` - 設定パーサーのテスト例
- `internal/scheduler/scheduler_test.go` - スケジューラーのテスト例

## 注意事項
- 統合テストは実行時間が長くなる可能性がある
- CI/CD環境での実行時間を考慮
- テストの並列実行を考慮した設計
