# deadman-go Design Doc

## 1. 概要

### 1.1 背景

deadman は `ping` を用いて複数ホストの疎通を監視し、curses ベースの UI で状態を可視化するシンプルな死活監視ツールである。

一方で、以下の課題・限界がある。

- Python 実装のため、単一バイナリ配布や小さいフットプリントでの配布がしづらい
- 並列処理や高スケールな監視において、設計を見直す余地がある
- Prometheus メトリクスや Grafana など、近年一般的な監視基盤との統合は標準機能として存在しない

そこで、Go 言語で deadman を再実装した **deadman-go** を開発し、

- 元 deadman の UX / 設定互換を維持しつつ
- Go ならではの軽量・高並列・単一バイナリ
- 将来的な Prometheus / Grafana 連携

を実現する。

### 1.2 目的

- 既存 deadman の機能をほぼ互換で提供する Go 製ツールを提供する
- Go の goroutine / channel による効率的な並列 ping 監視を実現する
- 将来的に Prometheus メトリクスを提供し、Grafana ダッシュボードで可視化できる足場を用意する

### 1.3 スコープ（初期リリース v0.1）

**含める**

- deadman.conf との互換性を意識した設定ファイル読み込み
- ICMP echo（ping）によるホスト疎通監視
- curses もしくは類似 TUI による監視画面（シンプルなステータス＋RTT 表示）
- goroutine によるホスト監視の並列化
- SIGHUP による設定ファイルリロード（可能であれば）

**含めない（後続リリースで対応）**

- Prometheus メトリクスエクスポート
- Grafana ダッシュボード定義
- 高度なアラート連携（メール／Slack など）
- SSH リレー（`relay=`）機能
- Windows / macOS への正式対応（初期リリースは Linux のみ）

---

## 2. 既存 deadman の機能整理

元 deadman（Python版）の主な機能は以下。

- **ICMP echo によるホスト死活監視**
  - 「ホスト名ラベル」と「IPアドレス」を列挙した config (`deadman.conf`) を読み込む
- **curses UI**
  - 各ターゲットの状態と RTT をバーグラフ形式で表示
- **設定ファイル形式**
  - `ラベル アドレス` の1行1エントリ
  - `---` によるグルーピング
  - ping オプションや SSH 経由での ping などを設定可能（`relay=`, `user=`, `key=` 等）
- **非同期モード**
  - async モードによる非同期 ping 送信
- **その他**
  - RTT バーのスケール変更
  - SIGHUP による config リロード（履歴保持）

deadman-go では、上記のうち **コアとなる監視ロジックと設定フォーマット互換** を優先的にサポートする。

---

## 3. deadman-go のゴール / 非ゴール

### 3.1 ゴール

1. **シンプルで依存の少ない単一バイナリ**
   - Go による statically linked バイナリを提供
2. **高並列監視**
   - 100〜1000 ホスト程度を実用的に監視できる並列モデル
3. **設定互換性**
   - 既存 `deadman.conf` を大きく変更せずに移行可能
4. **将来のメトリクス拡張**
   - Prometheus Exporter として動作できる構造（監視ループと HTTP `/metrics` の分離）

### 3.2 非ゴール（現時点）

- フル互換のキー/オプションの完全サポート（SSH 経由 ping などは段階的に）
- 大規模分散システム向けのフル機能監視製品
- SNMP / HTTP / TCP チェックなど、ICMP 以外の監視方式

---

## 4. ユースケース

- イベントネットワーク（会場ネットワーク等）での一時的な疎通監視
- 小〜中規模ネットワークの簡易死活監視
- 他監視システムの補助的な「ライブ感のある TUI」モニタ

---

## 5. 要求仕様

### 5.1 機能要件

1. **ターゲット定義**
   - 設定ファイルにより複数ホストを定義
   - `ラベル` `アドレス` 形式
   - `---` によるグループ区切り対応
2. **監視**
   - ICMP echo による死活判定
   - RTT を計測し、TUI 上で表示
   - 連続失敗回数によるステータス変化（例：OK / WARN / DOWN）
3. **UI**
   - curses 互換の TUI ライブラリ（tcell / bubbletea など）でステータス表示
   - グループごとに区切り線を表示
4. **設定リロード**
   - SIGHUP 受信時に設定ファイルを再読み込み
   - 可能なら既存ターゲットの履歴を維持
5. **ログ**
   - 標準出力 / ファイルに構造化ログを出力（JSON も検討）

### 5.2 非機能要件

- **性能**
  - 100〜1000ホスト程度を 1〜5秒間隔で監視できる（ホスト数 × インターバルはチューニング可能）
- **可搬性**
  - 初期リリース (v0.1) の公式サポート対象は Linux (x86_64/arm64) のみ
  - コンテナイメージ配布（Alpine + deadman-go など）を前提
  - macOS は v0.2 以降の対応候補とし、ICMP 実装の動作検証後にサポート可否を判断する
- **安定性**
  - ping ストームを防ぐためのレート制限
  - ICMP 権限（CAP_NET_RAW が無い場合は OS の ping コマンド利用にフォールバック）

---

## 6. アーキテクチャ概要

### 6.1 全体構成

- 単一バイナリ `deadman-go`
- 主な内部コンポーネント：
  - `config` パーサ
  - `scheduler`（監視頻度の制御）
  - `ping engine`（ICMP 実行）
  - `state store`（ターゲット状態管理）
  - `ui`（TUI レンダリング）
  - （将来）`metrics`（Prometheus Exporter）

### 6.2 Ping 実装方針

**初期案 (v0.1)**

- Linux を前提に、Go 製 ICMP ライブラリを利用して ICMP Echo を送信
- `--use-external-ping` オプションで外部 `ping` 実行モードも将来的に提供可能な構造とする

### 6.3 並列処理モデル

- 各ターゲットごとに **1 goroutine** の監視ループを持つ
- 全 goroutine を束ねる `scheduler` が tick（例: 1秒）ごとに「今 ping すべきターゲット」を判断
- グローバルで「同時実行 ping 数」の上限を設定（`max_concurrency`）し、`semaphore (chan struct{})` で制御
- `context.Context` によるキャンセル伝播（終了、リロード時）

---

## 7. 設定ファイル仕様（deadman.conf）

### 7.1 フォーマット概要

入力は元 deadman と同じ `deadman.conf` のフォーマットを前提とする。

- ホスト定義行:
  - `Name Address [key=value ...]`
  - 例:

    ```text
    google      216.58.197.174
    googleDNS   8.8.8.8
    ---
    kame        203.178.141.194
    ```

- コメント行:
  - `#` で始まる行はコメントとして扱う
  - うち、`# deadman-go:` で始まる行は deadman-go 独自のグローバル設定ディレクティブとして解釈する
    - 例:

      ```text
      # deadman-go: metrics.mode=per-target max_concurrency=200 interval=1s timeout=1s
      ```

- グループ区切り:
  - `---` をグループ区切りとして扱い、以降のターゲットに `Group` 名を付与する

### 7.2 deadman-go 独自グローバル設定

`# deadman-go:` 行にスペース区切りで `key=value` を並べる。

サポートするキー（v0.1 〜 将来拡張を想定）:

- `interval` : `time.Duration` 形式の監視間隔（例: `1s`, `500ms`）
- `timeout` : ping の timeout
- `max_concurrency` : 同時に発行する ping の最大数
- `metrics.mode` : `per-target` / `aggregated` / `both`
- `metrics.listen` : Prometheus の `/metrics` を公開する HTTP アドレス（例: `:9100`）
- `ui.scale` : RTT バーのスケール (ms)
- `ui.disable` : `true` の場合 TUI 無効（ログのみ）

#### 設定例

```text
#
# deadman config
#

# deadman-go: metrics.mode=per-target max_concurrency=200 interval=1s timeout=1s

google      216.58.197.174
googleDNS   8.8.8.8
---
kame        203.178.141.194
kame6       2001:200:dff:fff1:216:3eff:feb1:44d7
```

### 7.3 CLI オプションとの優先順位

- deadman.conf 内 `# deadman-go:` → **デフォルト値**
- CLI オプション → **config より強い（優先）**

例：

```bash
deadman-go --max-concurrency 50 deadman.conf
# → config の max_concurrency=200 より 50 が優先される
```

---

## 8. コンポーネント設計

### 8.1 型定義イメージ

```go
// メトリクス粒度
type MetricsMode string

const (
    MetricsModePerTarget  MetricsMode = "per-target"
    MetricsModeAggregated MetricsMode = "aggregated"
    MetricsModeBoth       MetricsMode = "both"
)

// deadman-go 独自のグローバル設定
type GlobalOptions struct {
    Interval       time.Duration // ping 間隔（ターゲットごと）
    Timeout        time.Duration // ping の timeout
    MaxConcurrency int           // 同時 ping 数上限

    MetricsMode   MetricsMode // per-target / aggregated / both
    MetricsListen string      // ":9100" など（未設定ならメトリクス無効）

    UIScale   int  // RTT バーのスケール (ms)
    UIDisable bool // true の場合 TUI 無効（ログのみ）
}

// deadman.conf の 1 ターゲット
type TargetConfig struct {
    Name    string            // 表示名（ラベル）
    Address string            // ping 宛先
    Group   string            // グループ名 or group index 文字列
    Options map[string]string // relay=, user= などの追加オプション（将来用）
}

// ファイル全体
type Config struct {
    Targets []TargetConfig
    Global  GlobalOptions
}

// CLI から渡す上書き値（nil なら「指定なし」扱い）
type CLIOverrides struct {
    Interval       *time.Duration
    Timeout        *time.Duration
    MaxConcurrency *int
    MetricsMode    *MetricsMode
    MetricsListen  *string
    UIDisable      *bool
}
```

### 8.2 Config パーサ疑似コード

```go
func LoadConfig(path string, cli CLIOverrides) (*Config, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    cfg := &Config{
        Global: defaultGlobalOptions(),
    }

    scanner := bufio.NewScanner(file)

    var groupIndex int
    var currentGroup string // 例: "group-0", "group-1" など

    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }

        // コメント
        if strings.HasPrefix(line, "#") {
            if strings.HasPrefix(line, "# deadman-go:") {
                parseDeadmanGoDirective(&cfg.Global, line)
            }
            continue
        }

        // グループ区切り
        if line == "---" {
            groupIndex++
            currentGroup = fmt.Sprintf("group-%d", groupIndex)
            continue
        }

        // ホスト定義行
        target, err := parseTargetLine(line, currentGroup)
        if err != nil {
            return nil, err
        }
        cfg.Targets = append(cfg.Targets, target)
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    applyCLIOverrides(&cfg.Global, cli)

    return cfg, nil
}

func defaultGlobalOptions() GlobalOptions {
    return GlobalOptions{
        Interval:       1 * time.Second,
        Timeout:        1 * time.Second,
        MaxConcurrency: 100,

        MetricsMode:   MetricsModePerTarget,
        MetricsListen: "", // 空ならメトリクス無効

        UIScale:   10,
        UIDisable: false,
    }
}
```

#### `# deadman-go:` 行のパース

```go
func parseDeadmanGoDirective(global *GlobalOptions, line string) {
    // line 例:
    // "# deadman-go: metrics.mode=per-target max_concurrency=200 interval=1s timeout=1s"
    s := strings.TrimSpace(strings.TrimPrefix(line, "#"))
    s = strings.TrimSpace(strings.TrimPrefix(s, "deadman-go:"))

    if s == "" {
        return
    }

    // スペース区切りで "key=value" を並べる想定
    tokens := strings.Fields(s)
    for _, tok := range tokens {
        kv := strings.SplitN(tok, "=", 2)
        if len(kv) != 2 {
            continue
        }
        key := kv[0]
        val := kv[1]

        switch key {
        case "interval":
            if d, err := time.ParseDuration(val); err == nil {
                global.Interval = d
            }
        case "timeout":
            if d, err := time.ParseDuration(val); err == nil {
                global.Timeout = d
            }
        case "max_concurrency":
            if n, err := strconv.Atoi(val); err == nil {
                global.MaxConcurrency = n
            }
        case "metrics.mode":
            switch val {
            case "per-target":
                global.MetricsMode = MetricsModePerTarget
            case "aggregated":
                global.MetricsMode = MetricsModeAggregated
            case "both":
                global.MetricsMode = MetricsModeBoth
            }
        case "metrics.listen":
            global.MetricsListen = val
        case "ui.scale":
            if n, err := strconv.Atoi(val); err == nil {
                global.UIScale = n
            }
        case "ui.disable":
            if b, err := strconv.ParseBool(val); err == nil {
                global.UIDisable = b
            }
        }
    }
}
```

#### ホスト定義行のパース

```go
func parseTargetLine(line string, group string) (TargetConfig, error) {
    // 例:
    // "google 216.58.197.174"
    // "relay1 192.0.2.1 relay=jump1 user=foo"
    fields := strings.Fields(line)
    if len(fields) < 2 {
        return TargetConfig{}, fmt.Errorf("invalid target line: %q", line)
    }

    name := fields[0]
    addr := fields[1]

    opts := make(map[string]string)
    if len(fields) > 2 {
        for _, f := range fields[2:] {
            kv := strings.SplitN(f, "=", 2)
            if len(kv) != 2 {
                continue
            }
            opts[kv[0]] = kv[1]
        }
    }

    return TargetConfig{
        Name:    name,
        Address: addr,
        Group:   group, // "" の場合もあり
        Options: opts,
    }, nil
}
```

#### CLI の上書き適用

```go
func applyCLIOverrides(global *GlobalOptions, cli CLIOverrides) {
    if cli.Interval != nil {
        global.Interval = *cli.Interval
    }
    if cli.Timeout != nil {
        global.Timeout = *cli.Timeout
    }
    if cli.MaxConcurrency != nil {
        global.MaxConcurrency = *cli.MaxConcurrency
    }
    if cli.MetricsMode != nil {
        global.MetricsMode = *cli.MetricsMode
    }
    if cli.MetricsListen != nil {
        global.MetricsListen = *cli.MetricsListen
    }
    if cli.UIDisable != nil {
        global.UIDisable = *cli.UIDisable
    }
}
```

---

## 9. Ping / State / Scheduler / UI

### 9.1 Pinger インターフェイス

```go
type Pinger interface {
    Ping(ctx context.Context, addr string, timeout time.Duration) (rtt time.Duration, err error)
}
```

実装例:

- `ICMPPinger`（Go の ICMP ライブラリ利用）
- （将来）`ExternalPinger`（`ping` コマンド実行）

### 9.2 State Store

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

    History []RTTPoint // TUI / メトリクス用に直近 N 件を保持
}

type StateStore struct {
    mu      sync.RWMutex
    targets map[string]*TargetStatus // key: Name or Address
}
```

主なメソッド:

- `UpdateResult(name string, rtt time.Duration, err error)`
- `GetSnapshot() []TargetStatus`
- `UpdateTargets([]TargetConfig)`（リロード時）

### 9.3 Scheduler

```go
type SchedulerConfig struct {
    Global GlobalOptions
    Pinger Pinger
    State  *StateStore
}

type Scheduler struct {
    cfg SchedulerConfig
    // セマフォ・ターゲットごとの goroutine 管理など
}

func NewScheduler(cfg SchedulerConfig) *Scheduler {
    return &Scheduler{cfg: cfg}
}

func (s *Scheduler) Run(ctx context.Context) error {
    // ターゲットごとに goroutine を立てて、
    // cfg.Global.Interval ごとに ping を打つループを回すイメージ。
    // ctx.Done() を見て終了。
    return nil
}

func (s *Scheduler) UpdateConfig(global GlobalOptions, targets []TargetConfig) {
    // 新しい設定を反映するロジック:
    // - Interval / Timeout / MaxConcurrency の更新
    // - 追加/削除されたターゲットの goroutine 管理 etc.
}
```

### 9.4 UI

- TUI ライブラリ: `bubbletea` / `tcell` などを想定
- 表示要素:
  - 上段: タイムスタンプ、全体統計（OK / WARN / DOWN ホスト数）
  - 中段: グループごとのターゲット一覧（ラベル / アドレス / 状態 / RTT＋バー）
  - 下段: ヘルプ (q: quit, r: reload, +/-: scale 調整 など)

```go
type UIConfig struct {
    Global GlobalOptions
    State  *StateStore
}

type UI struct {
    cfg UIConfig
}

func NewUI(cfg UIConfig) *UI {
    return &UI{cfg: cfg}
}

func (u *UI) Run(ctx context.Context) error {
    // ctx.Done() まで TUI イベントループを回し、
    // StateStore のスナップショットを定期的に読み取って描画する。
    return nil
}
```

---

## 10. Prometheus & Grafana（将来拡張）

### 10.1 メトリクス仕様

将来、以下のようなメトリクスを `/metrics` で公開する HTTP サーバを追加する。  
粒度は設定により切り替え可能とする。

**per-target モード:**

- `deadman_ping_rtt_seconds{target="google", group="internet"}`
  - 直近の RTT（Gauge）
- `deadman_ping_success_total{target="google"}`
- `deadman_ping_failure_total{target="google"}`
- `deadman_ping_up{target="google"}`
  - UP=1, DOWN=0（Gauge）

**aggregated モード:**

- `deadman_ping_rtt_seconds_avg{group="internet"}`
- `deadman_ping_targets_up{group="internet"}`
- `deadman_ping_targets_total{group="internet"}`

Config（deadman.conf）の `# deadman-go:` 行で  
`metrics.mode = "per-target" | "aggregated" | "both"` を指定し、  
デプロイ規模や Prometheus サーバの負荷に応じて切り替えられるようにする。

### 10.2 Grafana ダッシュボード案

- パネル例:
  - 「ターゲット別 RTT の時系列グラフ」
  - 「UP / DOWN 状態のステータステーブル」
  - 「失敗率ランキング」

**注意:** 初期リリースでは HTTP サーバ / メトリクスの実装は行わず、  
設計上の拡張ポイントとしてインターフェースのみ検討しておく。

---

## 11. CLI インターフェース案

```bash
deadman-go [OPTIONS] <config-file>

Options:
  -i, --interval duration     Ping interval per target (override config)
  -t, --timeout  duration     Ping timeout (override config)
      --max-concurrency int   Max concurrent pings (override config)
      --metrics-mode string   Metrics mode: per-target|aggregated|both
      --metrics-listen string Prometheus metrics listen addr (e.g. ":9100")
      --no-ui                 Run without TUI (log only)
  -v, --version               Show version
  -h, --help                  Show help
```

---

## 12. main.go のフロー疑似コード

```go
func main() {
    // 1. CLI フラグ
    var (
        flagInterval       = flag.Duration("interval", 0, "ping interval per target (override config)")
        flagTimeout        = flag.Duration("timeout", 0, "ping timeout (override config)")
        flagMaxConcurrency = flag.Int("max-concurrency", 0, "max concurrent pings (override config)")
        flagMetricsMode    = flag.String("metrics-mode", "", "metrics mode: per-target|aggregated|both")
        flagMetricsListen  = flag.String("metrics-listen", "", "metrics listen address (e.g. :9100)")
        flagNoUI           = flag.Bool("no-ui", false, "disable TUI")
        flagVersion        = flag.Bool("version", false, "show version")
    )

    flag.Parse()

    if *flagVersion {
        fmt.Println("deadman-go version x.y.z")
        return
    }

    // config path は位置引数で受ける想定
    args := flag.Args()
    if len(args) < 1 {
        fmt.Fprintln(os.Stderr, "usage: deadman-go [options] <config-file>")
        os.Exit(1)
    }
    configPath := args[0]

    // 2. CLI 上書き構造体を組み立て
    overrides := CLIOverrides{}
    if flagInterval != nil && *flagInterval > 0 {
        overrides.Interval = flagInterval
    }
    if flagTimeout != nil && *flagTimeout > 0 {
        overrides.Timeout = flagTimeout
    }
    if flagMaxConcurrency != nil && *flagMaxConcurrency > 0 {
        overrides.MaxConcurrency = flagMaxConcurrency
    }
    if flagMetricsMode != nil && *flagMetricsMode != "" {
        mode := MetricsMode(*flagMetricsMode)
        overrides.MetricsMode = &mode
    }
    if flagMetricsListen != nil && *flagMetricsListen != "" {
        overrides.MetricsListen = flagMetricsListen
    }
    if flagNoUI != nil && *flagNoUI {
        overrides.UIDisable = flagNoUI
    }

    // 3. config 読み込み
    cfg, err := LoadConfig(configPath, overrides)
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to load config: %v
", err)
        os.Exit(1)
    }

    // 4. コンポーネントを組み立てる

    // Ping 実装（v0.1 は Linux ICMP 前提）
    pinger, err := NewICMPPinger()
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to init pinger: %v
", err)
        os.Exit(1)
    }

    // 状態管理
    state := NewStateStore(cfg.Targets)

    // スケジューラ（goroutine 内で各ターゲット監視）
    scheduler := NewScheduler(SchedulerConfig{
        Global: cfg.Global,
        Pinger: pinger,
        State:  state,
    })

    // 5. コンテキスト + シグナルハンドリング
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // SIGHUP 用のハンドラは別 goroutine で扱う（後述）

    // 6. メトリクス HTTP サーバ（将来的に）
    if cfg.Global.MetricsListen != "" {
        go func() {
            if err := StartMetricsServer(cfg.Global.MetricsListen, state); err != nil {
                fmt.Fprintf(os.Stderr, "metrics server error: %v
", err)
            }
        }()
    }

    // 7. UI 起動
    if !cfg.Global.UIDisable {
        ui := NewUI(UIConfig{
            Global: cfg.Global,
            State:  state,
        })

        go func() {
            if err := ui.Run(ctx); err != nil {
                fmt.Fprintf(os.Stderr, "ui error: %v
", err)
            }
        }()
    }

    // 8. SIGHUP で設定リロード
    go func() {
        hupCh := make(chan os.Signal, 1)
        signal.Notify(hupCh, syscall.SIGHUP)
        defer signal.Stop(hupCh)

        for {
            select {
            case <-ctx.Done():
                return
            case <-hupCh:
                // 設定再読込
                newCfg, err := LoadConfig(configPath, overrides)
                if err != nil {
                    fmt.Fprintf(os.Stderr, "failed to reload config: %v
", err)
                    continue
                }
                scheduler.UpdateConfig(newCfg.Global, newCfg.Targets)
                state.UpdateTargets(newCfg.Targets)
            }
        }
    }()

    // 9. スケジューラ実行（blocking）
    if err := scheduler.Run(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "scheduler error: %v
", err)
        os.Exit(1)
    }
}
```

---

## 13. SSH Relay（将来拡張）

v0.1 では SSH 経由 ping (`relay=...`) をサポートしない。

将来的に以下の方針で実装を検討する：

- `golang.org/x/crypto/ssh` による SSH セッション確立
- リレー先で `ping -c 1 <target>` などのコマンドを実行し、標準出力から RTT と成功 / 失敗をパース
- リレー先ごとにコネクションプールを持ち、接続の張り直しコストを抑制
- ping コマンドのフォーマット差異（Linux / BSD / BusyBox 等）を吸収するパーサを実装

これらにより、既存 deadman の `relay=` 設定に近い機能を提供するが、  
実装コストと運用負荷の観点から v0.2 以降の experimental 機能として扱う。

---

## 14. リリース計画

### v0.1 (MVP)

- Go 実装のコア:
  - Config パース
  - Go ICMP ライブラリによる ping
  - goroutine による並列監視
  - シンプルな TUI 表示
- Linux 向け単一バイナリ & コンテナイメージ
- deadman.conf 互換（ホスト定義 / グループ / コメント）

### v0.2

- SIGHUP / 設定リロードの強化
- RTT スケールの動的変更
- 外部 ping コマンドモード
- macOS 対応検証

### v0.3

- Prometheus メトリクスエクスポート
- Grafana ダッシュボード JSON のサンプル提供
- SSH Relay の experimental 実装

---

## 15. リスク・懸念点

- ICMP 実行に必要な権限（CAP_NET_RAW など）の取り扱い
- 大量ターゲット監視でのリソース使用量（goroutine, FD, CPU）
- TUI ライブラリ選定（長期のメンテ性）
- 既存 deadman と期待挙動の微妙な差異
- SSH Relay 実装時の複雑さ（接続数管理、エラー原因の切り分けなど）

---

以上を deadman-go の Design / Spec として利用する。
