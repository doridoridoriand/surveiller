package ui

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/state"
	"github.com/gdamore/tcell/v2"
)

const (
	uiRefreshInterval = 500 * time.Millisecond
	minBoxHeight      = 4
)

// UI renders a TUI view of target status.
type UI struct {
	cfg   config.GlobalOptions
	state state.Store
}

// New returns a UI instance.
func New(cfg config.GlobalOptions, store state.Store) *UI {
	return &UI{cfg: cfg, state: store}
}

// Run blocks until the context is cancelled or the user quits.
func (u *UI) Run(ctx context.Context) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	screen.HideCursor()
	defer screen.Fini()

	eventCh := make(chan tcell.Event, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			ev := screen.PollEvent()
			if ev == nil {
				return
			}
			select {
			case eventCh <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()

	ticker := time.NewTicker(uiRefreshInterval)
	defer ticker.Stop()

	u.render(screen, u.state.GetSnapshot())
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev := <-eventCh:
			switch ev := ev.(type) {
			case *tcell.EventKey:
				if ev.Key() == tcell.KeyCtrlC || ev.Rune() == 'q' {
					return context.Canceled
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		case <-ticker.C:
			u.render(screen, u.state.GetSnapshot())
		}
	}
}

func (u *UI) render(screen tcell.Screen, snapshot []state.TargetStatus) {
	screen.Clear()
	width, height := screen.Size()
	if width < 20 || height < 5 {
		screen.Show()
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	header := fmt.Sprintf(" deadman-go  %s  (q to quit)", now)
	drawText(screen, 0, 0, width, header, tcell.StyleDefault.Bold(true))

	// 設定情報を2行目に表示
	configInfo := formatConfigInfo(u.cfg)
	drawText(screen, 0, 1, width, configInfo, tcell.StyleDefault.Foreground(tcell.ColorGray))

	groups := groupTargets(snapshot)
	y := 2
	for _, group := range groups {
		if height-y < minBoxHeight {
			break
		}
		boxHeight := len(group.Targets) + 3
		if boxHeight > height-y {
			boxHeight = height - y
		}
		u.drawGroupBox(screen, 0, y, width, boxHeight, group)
		y += boxHeight
	}

	screen.Show()
}

type targetGroup struct {
	Name    string
	Targets []state.TargetStatus
}

func groupTargets(snapshot []state.TargetStatus) []targetGroup {
	if len(snapshot) == 0 {
		return nil
	}
	groups := make(map[string][]state.TargetStatus)
	for _, target := range snapshot {
		name := strings.TrimSpace(target.Group)
		if name == "" {
			name = "default"
		}
		groups[name] = append(groups[name], target)
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == "default" {
			return true
		}
		if names[j] == "default" {
			return false
		}
		return names[i] < names[j]
	})

	result := make([]targetGroup, 0, len(names))
	for _, name := range names {
		targets := groups[name]
		sort.Slice(targets, func(i, j int) bool {
			return targets[i].Name < targets[j].Name
		})
		result = append(result, targetGroup{Name: name, Targets: targets})
	}
	return result
}

func (u *UI) drawGroupBox(screen tcell.Screen, x, y, width, height int, group targetGroup) {
	drawBox(screen, x, y, width, height)

	title := fmt.Sprintf(" %s ", group.Name)
	drawText(screen, x+2, y, width-4, title, tcell.StyleDefault.Bold(true))

	if height <= 2 {
		return
	}

	rowY := y + 1
	maxRows := height - 2
	for i := 0; i < len(group.Targets) && i < maxRows; i++ {
		target := group.Targets[i]
		line := u.formatTargetLine(width-2, target)
		drawStyledText(screen, x+1, rowY+i, width-2, line)
	}
}

func (u *UI) formatTargetLine(width int, target state.TargetStatus) []styledRune {
	statusStyle := statusStyle(target.Status)
	name := padOrTrim(target.Name, minInt(14, width))
	addr := padOrTrim(target.Address, minInt(18, width))
	status := padOrTrim(string(target.Status), 6)

	// 平均RTTを計算
	avgRTT := calculateAvgRTT(target)
	rttLabel := "RTT:"
	rtt := padOrTrim(fmt.Sprintf("%s%s", rttLabel, formatRTT(avgRTT)), 12)

	// LOSS率を計算して表示
	lossPercent := calculateLossPercent(target)
	loss := padOrTrim(fmt.Sprintf("LOSS:%.1f%%", lossPercent), 12)

	parts := []styledText{
		{text: name, style: tcell.StyleDefault},
		{text: " ", style: tcell.StyleDefault},
		{text: addr, style: tcell.StyleDefault},
		{text: " ", style: tcell.StyleDefault},
		{text: status, style: statusStyle},
		{text: " ", style: tcell.StyleDefault},
		{text: rtt, style: tcell.StyleDefault},
		{text: " ", style: tcell.StyleDefault},
		{text: loss, style: statusStyle},
		{text: " ", style: tcell.StyleDefault},
	}

	used := 0
	for _, p := range parts {
		used += len([]rune(p.text))
	}
	barWidth := width - used
	if barWidth > 0 {
		bar := buildBar(target, u.cfg.UIScale, barWidth)
		parts = append(parts, styledText{text: bar, style: statusStyle})
	}

	return flattenStyledText(parts, width)
}

func buildBar(target state.TargetStatus, scale int, width int) string {
	if width <= 0 {
		return ""
	}
	if scale <= 0 {
		scale = 10
	}
	ms := float64(target.LastRTT.Milliseconds())
	if ms <= 0 {
		return strings.Repeat(" ", width)
	}
	units := int(math.Round(ms / float64(scale)))
	if units > width {
		units = width
	}
	if units < 0 {
		units = 0
	}
	return strings.Repeat("#", units) + strings.Repeat(" ", width-units)
}

func drawBox(screen tcell.Screen, x, y, width, height int) {
	if width < 2 || height < 2 {
		return
	}
	right := x + width - 1
	bottom := y + height - 1

	setCell(screen, x, y, '+', tcell.StyleDefault)
	setCell(screen, right, y, '+', tcell.StyleDefault)
	setCell(screen, x, bottom, '+', tcell.StyleDefault)
	setCell(screen, right, bottom, '+', tcell.StyleDefault)

	for col := x + 1; col < right; col++ {
		setCell(screen, col, y, '-', tcell.StyleDefault)
		setCell(screen, col, bottom, '-', tcell.StyleDefault)
	}
	for row := y + 1; row < bottom; row++ {
		setCell(screen, x, row, '|', tcell.StyleDefault)
		setCell(screen, right, row, '|', tcell.StyleDefault)
	}
}

func drawText(screen tcell.Screen, x, y, width int, text string, style tcell.Style) {
	drawStyledText(screen, x, y, width, []styledRune{{r: []rune(text), style: style}})
}

type styledText struct {
	text  string
	style tcell.Style
}

type styledRune struct {
	r     []rune
	style tcell.Style
}

func drawStyledText(screen tcell.Screen, x, y, width int, parts []styledRune) {
	if width <= 0 {
		return
	}
	col := x
	for _, part := range parts {
		for _, r := range part.r {
			if col >= x+width {
				return
			}
			setCell(screen, col, y, r, part.style)
			col++
		}
	}
	for col < x+width {
		setCell(screen, col, y, ' ', tcell.StyleDefault)
		col++
	}
}

func flattenStyledText(parts []styledText, width int) []styledRune {
	result := make([]styledRune, 0, len(parts))
	used := 0
	for _, part := range parts {
		runes := []rune(part.text)
		if used+len(runes) > width {
			runes = runes[:maxInt(0, width-used)]
		}
		result = append(result, styledRune{r: runes, style: part.style})
		used += len(runes)
		if used >= width {
			break
		}
	}
	return result
}

func setCell(screen tcell.Screen, x, y int, r rune, style tcell.Style) {
	screen.SetContent(x, y, r, nil, style)
}

func padOrTrim(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) > width {
		return string(runes[:width])
	}
	if len(runes) < width {
		return value + strings.Repeat(" ", width-len(runes))
	}
	return value
}

func formatRTT(rtt time.Duration) string {
	if rtt <= 0 {
		return "-"
	}
	if rtt < time.Millisecond {
		return fmt.Sprintf("%dus", rtt.Microseconds())
	}
	if rtt < time.Second {
		return fmt.Sprintf("%dms", rtt.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", rtt.Seconds())
}

func calculateAvgRTT(target state.TargetStatus) time.Duration {
	if len(target.History) == 0 {
		return target.LastRTT
	}
	var sum time.Duration
	for _, point := range target.History {
		sum += point.RTT
	}
	return sum / time.Duration(len(target.History))
}

func calculateLossPercent(target state.TargetStatus) float64 {
	total := target.TotalSuccess + target.TotalFailure
	if total == 0 {
		return 0.0
	}
	return float64(target.TotalFailure) / float64(total) * 100.0
}

func statusStyle(status state.Status) tcell.Style {
	switch status {
	case state.StatusOK:
		return tcell.StyleDefault.Foreground(tcell.ColorGreen)
	case state.StatusWarn:
		return tcell.StyleDefault.Foreground(tcell.ColorYellow)
	case state.StatusDown:
		return tcell.StyleDefault.Foreground(tcell.ColorRed)
	default:
		return tcell.StyleDefault.Foreground(tcell.ColorGray)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func formatConfigInfo(cfg config.GlobalOptions) string {
	intervalStr := formatDuration(cfg.Interval)
	timeoutStr := formatDuration(cfg.Timeout)
	return fmt.Sprintf(" interval=%s  timeout=%s  max_concurrency=%d  ui.scale=%d",
		intervalStr, timeoutStr, cfg.MaxConcurrency, cfg.UIScale)
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
