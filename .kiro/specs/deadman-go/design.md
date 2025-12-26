# deadman-go 設計書

## 概要

deadman-go は、既存の Python 製 deadman ツールを Go 言語で再実装した死活監視ツールです。ICMP ping による複数ホストの疎通監視を行い、curses ベースの TUI で状態を可視化します。Go の goroutine と channel を活用した高並列処理により、100-1000 ホスト規模の監視を効率的に実行できます。

主な特徴：
- 既存 deadman.conf との互換性を維持
- Go による軽量・高並列・単一バイナリ配布
- リアルタイム TUI による状態可視化
- 将来的な Prometheus/Grafana 連携の基盤

## アーキテクチャ

### 全体構成

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Config        │    │   Scheduler     │    │   State Store   │
│   Parser        │───▶│   (goroutine    │───▶│   (thread-safe) │
│                 │    │    manager)     │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        ▲
                                ▼                        │
┌─────────────────┐    ┌─────────────────┐              │
│   TUI           │    │   Ping Engine   │              │
│   (tcell)       │◀───│   (ICMP +       │──────────────┘
│                 │    │    Fallback)    │
└─────────────────┘    └─────────────────┘
```

### コンポーネント間の関係

1. **Config Parser**: deadman.conf ファイルを解析し、ターゲット定義とグローバル設定を抽出
2. **Scheduler**: 各ターゲットの監視 goroutine を管理し、並列度を制御
3. **Ping Engine**: ICMP echo request/reply を処理し、RTT を計測。権限エラー時は外部コマンドにフォールバック
4. **State Store**: 各ターゲットの状態（RTT、成功/失敗履歴、ステータス）を管理
5. **TUI**: リアルタイムで監視状態を表示するターミナルインターフェース（tcellライブラリ使用）

## コンポーネントと インターフェース

### Config Parser

```go
type GlobalOptions struct {
    Interval       time.Duration // ping 間隔
    Timeout        time.Duration // ping タイムアウト
    MaxConcurrency int           // 同時 ping 数上限
    MetricsMode    MetricsMode   // メトリクス出力モード
    MetricsListen  string        // メトリクス待受アドレス
    UIScale        int           // RTT バーのスケール (ms)
    UIDisable      bool          // TUI 無効フラグ
}

type TargetConfig struct {
    Name    string            // 表示名
    Address string            // ping 宛先
    Group   string            // グループ名
    Options map[string]string // 将来の拡張用オプション
}

type Config struct {
    Targets []TargetConfig
    Global  GlobalOptions
}

type CLIOverrides struct {
    Interval       *time.Duration
    Timeout        *time.Duration
    MaxConcurrency *int
    MetricsMode    *MetricsMode
    MetricsListen  *string
    UIDisable      *bool
}

type Parser interface {
    LoadConfig(path string, overrides CLIOverrides) (*Config, error)
    ParseDeadmanGoDirective(line string) (map[string]string, error)
    ParseTargetLine(line string, group string) (TargetConfig, error)
}
```

### Ping Engine

```go
type Result struct {
    RTT     time.Duration
    Success bool
    Error   error
}

type Pinger interface {
    Ping(ctx context.Context, addr string, timeout time.Duration) Result
}

type ICMPPinger struct {
    id  int
    seq uint32
}

type FallbackPinger struct {
    primary   Pinger
    secondary Pinger
}
```

### State Store

```go
type Status string

const (
    StatusUnknown Status = "UNKNOWN"
    StatusOK      Status = "OK"
    StatusWarn    Status = "WARN"
    StatusDown    Status = "DOWN"
)

type RTTPoint struct {
    Time time.Time
    RTT  time.Duration
}

type TargetStatus struct {
    Name          string
    Address       string
    Group         string
    LastRTT       time.Duration
    LastSuccessAt time.Time
    LastFailureAt time.Time
    ConsecutiveOK int
    ConsecutiveNG int
    Status        Status
    History       []RTTPoint
}

type Store interface {
    UpdateResult(name string, result Result)
    GetSnapshot() []TargetStatus
    UpdateTargets(targets []TargetConfig)
    GetTargetStatus(name string) (TargetStatus, bool)
}

type StoreImpl struct {
    mu            sync.RWMutex
    targets       map[string]*TargetStatus
    historySize   int
    downThreshold int
    timeout       time.Duration
}
```

### Scheduler

```go
type Scheduler interface {
    Run(ctx context.Context) error
    UpdateConfig(global GlobalOptions, targets []TargetConfig)
    Stop()
}

type Impl struct {
    mu         sync.RWMutex
    cfg        GlobalOptions
    targets    map[string]TargetConfig
    pinger     Pinger
    state      Store
    semaphore  chan struct{}
    targetJobs map[string]context.CancelFunc
    wg         sync.WaitGroup
    cancel     context.CancelFunc
    runCtx     context.Context
}
```

## データモデル

### ターゲット状態の遷移

```
UNKNOWN ──ping success──▶ OK/WARN (RTTに基づく)
   │                      │
   │                      │ consecutive failures > threshold
   │                      ▼
   └──ping failure──▶ WARN ──more failures──▶ DOWN
                       ▲                        │
                       │                        │
                       └──ping success──────────┘
```

状態判定ロジック：
- **成功時**: RTTがtimeoutの25%以内 → OK、50%以内 → WARN、50%超 → WARN
- **失敗時**: 連続失敗回数が閾値（デフォルト3回）未満 → WARN、以上 → DOWN

### RTT 履歴管理

各ターゲットは直近 N 件（デフォルト100件）の RTT データを保持：

```go
type RTTPoint struct {
    Time time.Time
    RTT  time.Duration
}

// RTT履歴はTargetStatus内のHistoryフィールドで管理
// リングバッファではなく、スライスによる実装
```

### 設定ファイル構造

```
# deadman-go: interval=1s timeout=1s max_concurrency=100 ui.scale=10

google      216.58.197.174
googleDNS   8.8.8.8
---
kame        203.178.141.194
kame6       2001:200:dff:fff1:216:3eff:feb1:44d7
```

サポートされるディレクティブ：
- `interval`: ping間隔
- `timeout`: pingタイムアウト
- `max_concurrency`: 最大並列数
- `metrics.mode`: メトリクスモード (per-target|aggregated|both)
- `metrics.listen`: メトリクス待受アドレス
- `ui.scale`: RTTバーのスケール (ms)
- `ui.disable`: TUI無効化

## 正確性プロパティ

*プロパティとは、システムの全ての有効な実行において真であるべき特性や動作のことです。これは人間が読める仕様と機械で検証可能な正確性保証の橋渡しとなります。*
### プロパティ反映

前作業分析を確認した結果、以下の冗長性を特定しました：

- CLI オプション処理のプロパティ（8.1-8.4）は、設定優先度の単一プロパティに統合可能
- ログ記録のプロパティ（6.1-6.4）は、構造化ログ出力の包括的プロパティに統合可能
- 設定リロードのプロパティ（5.2-5.3）は、動的設定更新の単一プロパティに統合可能

### 正確性プロパティ

**プロパティ 1: 設定ファイル解析の正確性**
*任意の* 有効な deadman.conf 形式のファイルに対して、パーサーは `ラベル アドレス` 形式の行を正しく TargetConfig に変換し、`---` 区切りに基づいて適切なグループを割り当てる
**検証対象: 要件 1.1, 1.2**

**プロパティ 2: グローバル設定ディレクティブの処理**
*任意の* `# deadman-go:` で始まる有効なディレクティブ行に対して、システムは key=value ペアを正しく解析して GlobalOptions に反映する
**検証対象: 要件 1.3**

**プロパティ 3: コメント行の無視**
*任意の* `#` で始まる通常のコメント行を含む設定ファイルに対して、コメント行はターゲット定義に影響を与えない
**検証対象: 要件 1.4**

**プロパティ 4: 設定ファイル構文エラー検出**
*任意の* 無効な形式の設定ファイルに対して、システムは構文エラーを検出して適切なエラーメッセージを返す
**検証対象: 要件 1.5**

**プロパティ 5: ping 結果の状態更新**
*任意の* ping 結果（成功または失敗）に対して、StateStore は RTT の記録、失敗カウントの更新、状態遷移を正しく実行する。成功時はRTTに基づいてOK/WARNを判定し（timeout の25%以内でOK、50%以内でWARN、50%超でもWARN）、失敗時は連続失敗回数に基づいてWARN/DOWNを判定する
**検証対象: 要件 2.2, 2.3, 2.4**

**プロパティ 6: タイムアウト処理**
*任意の* タイムアウト値を超える応答時間に対して、システムはそれを失敗として扱い、失敗カウントを増加させる
**検証対象: 要件 2.5**

**プロパティ 7: TUI グループ表示**
*任意の* グループ構成を持つターゲットリストに対して、TUI はグループごとに区切り線を正しく表示する
**検証対象: 要件 3.2**

**プロパティ 8: TUI RTT バーグラフ表示**
*任意の* RTT 値に対して、TUI は設定されたスケールに基づいて正しくバーグラフを描画する
**検証対象: 要件 3.3**

**プロパティ 9: TUI 状態更新の即時反映**
*任意の* ターゲット状態変化に対して、TUI は状態表示を即座に更新する
**検証対象: 要件 3.4**

**プロパティ 10: 並列監視の独立性**
*任意の* ターゲット数に対して、システムは各ターゲットを独立した goroutine で監視し、goroutine 数がターゲット数と一致する
**検証対象: 要件 4.1**

**プロパティ 11: 並列度制御**
*任意の* 同時実行上限設定に対して、システムは上限を超える ping 要求を適切に制御し、ping ストームを防ぐ
**検証対象: 要件 4.2**

**プロパティ 12: 監視間隔の遵守**
*任意の* 監視間隔設定に対して、スケジューラーは指定された間隔で各ターゲットの ping をスケジュールする
**検証対象: 要件 4.3**

**プロパティ 13: 適切な終了処理**
*任意の* 終了要求に対して、システムは全ての goroutine を適切に終了し、リソースを正しく解放する
**検証対象: 要件 4.4**

**プロパティ 14: 動的設定更新**
*任意の* 設定変更（ターゲットの追加・削除）に対して、システムは新しいターゲットの監視開始と削除されたターゲットの監視停止を正しく実行する
**検証対象: 要件 5.2, 5.3**

**プロパティ 15: 設定リロードエラー処理**
*任意の* 無効な設定ファイルでのリロード試行に対して、システムは既存の設定を維持し、適切なエラーログを出力する
**検証対象: 要件 5.4**

**プロパティ 16: 履歴保持**
*任意の* 設定リロードに対して、システムは共通するターゲットの RTT 履歴を可能な限り保持する
**検証対象: 要件 5.5**

**プロパティ 17: 構造化ログ出力**
*任意の* システムイベント（ping 結果、設定読み込み、エラー）に対して、システムは適切な構造化ログを出力し、ログレベル設定に従ってフィルタリングする
**検証対象: 要件 6.1, 6.2, 6.3, 6.4, 6.5**

**プロパティ 18: CLI 設定優先度**
*任意の* CLI オプションと設定ファイルの組み合わせに対して、システムは CLI オプションを設定ファイルより優先して適用する
**検証対象: 要件 8.1, 8.2, 8.3, 8.4**

## エラーハンドリング

### エラー分類と対応

1. **設定エラー**
   - 無効な設定ファイル形式
   - 存在しない設定ファイル
   - 権限不足による読み込み失敗
   - 対応: 詳細なエラーメッセージとヘルプ表示

2. **ネットワークエラー**
   - ICMP 権限不足
   - ネットワーク到達不可
   - DNS 解決失敗
   - 対応: ログ記録と状態更新、必要に応じてフォールバック

3. **システムエラー**
   - メモリ不足
   - ファイルディスクリプタ不足
   - シグナル処理エラー
   - 対応: 適切なログ記録と graceful shutdown

### エラー回復戦略

- **一時的エラー**: 指数バックオフによるリトライ
- **永続的エラー**: エラー状態の記録と継続監視
- **致命的エラー**: 適切なクリーンアップ後の終了

## テスト戦略

### 単体テスト

各コンポーネントの基本機能を検証：

- Config Parser: 有効/無効な設定ファイルの解析
- Ping Engine: モックを使用した ping 結果処理
- State Store: 状態更新と履歴管理
- Scheduler: goroutine 管理と並列度制御
- TUI: 表示ロジックとユーザー入力処理

### プロパティベーステスト

**使用ライブラリ**: `gopter` (Go 用プロパティベーステストライブラリ)

**設定**: 各プロパティテストは最低 100 回の反復実行

**プロパティテスト実装要件**:
- 各プロパティテストは設計書の正確性プロパティを実装する単一のテストとする
- テストコメントで対応するプロパティを明記: `**Feature: deadman-go, Property {number}: {property_text}**`
- スマートなジェネレータを使用して入力空間を適切に制約する

### 統合テスト

エンドツーエンドのシナリオを検証：

- 設定ファイル読み込みから監視開始まで
- 設定リロードによる動的更新
- シグナルハンドリングと終了処理
- TUI とバックエンドの連携

### テスト実行戦略

1. **実装優先**: 機能実装後に対応するテストを作成
2. **段階的検証**: コア機能から順次テスト追加
3. **継続的検証**: CI/CD パイプラインでの自動テスト実行