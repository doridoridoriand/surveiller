package metrics

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/doridoridoriand/surveiller/internal/config"
	"github.com/doridoridoriand/surveiller/internal/ping"
	"github.com/doridoridoriand/surveiller/internal/state"
)

type fakeStore struct {
	snapshot []state.TargetStatus
}

func (f fakeStore) UpdateResult(name string, result ping.Result) {}

func (f fakeStore) GetSnapshot() []state.TargetStatus {
	return f.snapshot
}

func (f fakeStore) UpdateTargets(targets []config.TargetConfig) {}

func (f fakeStore) GetTargetStatus(name string) (state.TargetStatus, bool) {
	return state.TargetStatus{}, false
}

func TestWriteAggregated(t *testing.T) {
	snapshot := []state.TargetStatus{
		{Status: state.StatusOK},
		{Status: state.StatusWarn},
		{Status: state.StatusDown},
		{Status: state.StatusUnknown},
	}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	writeAggregated(writer, snapshot)
	_ = writer.Flush()

	got := buf.String()
	expected := strings.Join([]string{
		"surveiller_targets_total 4",
		"surveiller_targets_ok 1",
		"surveiller_targets_warn 1",
		"surveiller_targets_down 1",
		"surveiller_targets_unknown 1",
		"",
	}, "\n")
	if got != expected {
		t.Fatalf("unexpected aggregated metrics:\n%s", got)
	}
}

func TestWritePerTarget(t *testing.T) {
	snapshot := []state.TargetStatus{
		{
			Name:    "name\"1",
			Address: "addr\\path",
			Group:   "grp",
			Status:  state.StatusOK,
			LastRTT: 15 * time.Millisecond,
		},
		{
			Name:    "down",
			Address: "1.1.1.1",
			Group:   "",
			Status:  state.StatusDown,
		},
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	writePerTarget(writer, snapshot)
	_ = writer.Flush()

	labels1 := `target="name\\\"1",address="addr\\\\path",group="grp"`
	labels2 := `target="down",address="1.1.1.1",group=""`
	expected := strings.Join([]string{
		"surveiller_target_up{" + labels1 + "} 1",
		"surveiller_target_rtt_ms{" + labels1 + "} 15",
		"surveiller_target_up{" + labels2 + "} 0",
		"",
	}, "\n")
	if buf.String() != expected {
		t.Fatalf("unexpected per-target metrics:\n%s", buf.String())
	}
}

func TestEscapeLabel(t *testing.T) {
	if got := escapeLabel(`value"slash\`); got != `value\"slash\\` {
		t.Fatalf("unexpected escaped label: %q", got)
	}
}

func TestHandlerMethodNotAllowed(t *testing.T) {
	server := NewServer(config.MetricsModeAggregated, fakeStore{})
	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
}

func TestHandlerAggregatedOutput(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}
	server := NewServer(config.MetricsModeAggregated, store)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "text/plain; version=0.0.4" {
		t.Fatalf("unexpected content type: %q", contentType)
	}
	if !strings.Contains(rec.Body.String(), "surveiller_targets_total 1") {
		t.Fatalf("expected aggregated metrics output, got %q", rec.Body.String())
	}
}

func TestServeContextCancellation(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Serve(ctx, "127.0.0.1:0", config.MetricsModeAggregated, store)
	if err == nil {
		t.Fatalf("expected context cancellation error")
	}
}
