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

func (f fakeStore) UpdateTimeout(timeout time.Duration) {}

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

// Test per-target mode metrics output
func TestHandlerPerTargetMode(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{
			{
				Name:    "test1",
				Address: "1.1.1.1",
				Group:   "group1",
				Status:  state.StatusOK,
				LastRTT: 10 * time.Millisecond,
			},
			{
				Name:    "test2",
				Address: "2.2.2.2",
				Group:   "group2",
				Status:  state.StatusDown,
				LastRTT: 0,
			},
		},
	}
	server := NewServer(config.MetricsModePerTarget, store)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	// Should contain per-target metrics
	if !strings.Contains(body, "surveiller_target_up{") {
		t.Fatalf("expected per-target up metrics, got %q", body)
	}
	if !strings.Contains(body, "surveiller_target_rtt_ms{") {
		t.Fatalf("expected per-target RTT metrics, got %q", body)
	}
	// Should NOT contain aggregated metrics
	if strings.Contains(body, "surveiller_targets_total") {
		t.Fatalf("should not contain aggregated metrics in per-target mode, got %q", body)
	}
}

// Test aggregated mode metrics output
func TestHandlerAggregatedMode(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{
			{Status: state.StatusOK},
			{Status: state.StatusWarn},
			{Status: state.StatusDown},
		},
	}
	server := NewServer(config.MetricsModeAggregated, store)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	// Should contain aggregated metrics
	if !strings.Contains(body, "surveiller_targets_total 3") {
		t.Fatalf("expected total targets count, got %q", body)
	}
	if !strings.Contains(body, "surveiller_targets_ok 1") {
		t.Fatalf("expected OK targets count, got %q", body)
	}
	if !strings.Contains(body, "surveiller_targets_warn 1") {
		t.Fatalf("expected WARN targets count, got %q", body)
	}
	if !strings.Contains(body, "surveiller_targets_down 1") {
		t.Fatalf("expected DOWN targets count, got %q", body)
	}
	// Should NOT contain per-target metrics
	if strings.Contains(body, "surveiller_target_up{") {
		t.Fatalf("should not contain per-target metrics in aggregated mode, got %q", body)
	}
}

// Test both mode metrics output
func TestHandlerBothMode(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{
			{
				Name:    "test1",
				Address: "1.1.1.1",
				Group:   "group1",
				Status:  state.StatusOK,
				LastRTT: 5 * time.Millisecond,
			},
		},
	}
	server := NewServer(config.MetricsModeBoth, store)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	// Should contain both aggregated and per-target metrics
	if !strings.Contains(body, "surveiller_targets_total 1") {
		t.Fatalf("expected aggregated metrics in both mode, got %q", body)
	}
	if !strings.Contains(body, "surveiller_target_up{") {
		t.Fatalf("expected per-target metrics in both mode, got %q", body)
	}
}

// Test empty mode (no metrics output)
func TestHandlerEmptyMode(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}
	server := NewServer("", store) // Empty mode
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	// Should be empty (no metrics)
	if strings.TrimSpace(body) != "" {
		t.Fatalf("expected empty metrics output for empty mode, got %q", body)
	}
}

// Test comprehensive label escaping functionality
func TestLabelEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no escaping needed",
			input:    "simple_value",
			expected: "simple_value",
		},
		{
			name:     "escape quotes",
			input:    `value"with"quotes`,
			expected: `value\"with\"quotes`,
		},
		{
			name:     "escape backslashes",
			input:    `value\with\backslashes`,
			expected: `value\\with\\backslashes`,
		},
		{
			name:     "escape both quotes and backslashes",
			input:    `value"with\both`,
			expected: `value\"with\\both`,
		},
		{
			name:     "multiple consecutive escapes",
			input:    `value\\""\\`,
			expected: `value\\\\\"\"\\\\`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeLabel(tt.input)
			if got != tt.expected {
				t.Errorf("escapeLabel(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// Test per-target metrics with various status combinations
func TestWritePerTargetVariousStatuses(t *testing.T) {
	snapshot := []state.TargetStatus{
		{
			Name:    "ok_target",
			Address: "1.1.1.1",
			Group:   "group1",
			Status:  state.StatusOK,
			LastRTT: 10 * time.Millisecond,
		},
		{
			Name:    "warn_target",
			Address: "2.2.2.2",
			Group:   "group1",
			Status:  state.StatusWarn,
			LastRTT: 50 * time.Millisecond,
		},
		{
			Name:    "down_target",
			Address: "3.3.3.3",
			Group:   "group2",
			Status:  state.StatusDown,
			LastRTT: 0,
		},
		{
			Name:    "unknown_target",
			Address: "4.4.4.4",
			Group:   "",
			Status:  state.StatusUnknown,
			LastRTT: 0,
		},
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	writePerTarget(writer, snapshot)
	_ = writer.Flush()

	output := buf.String()

	// Check that OK target has up=1
	if !strings.Contains(output, `surveiller_target_up{target="ok_target",address="1.1.1.1",group="group1"} 1`) {
		t.Errorf("expected OK target to have up=1, got %q", output)
	}

	// Check that non-OK targets have up=0
	if !strings.Contains(output, `surveiller_target_up{target="warn_target",address="2.2.2.2",group="group1"} 0`) {
		t.Errorf("expected WARN target to have up=0, got %q", output)
	}
	if !strings.Contains(output, `surveiller_target_up{target="down_target",address="3.3.3.3",group="group2"} 0`) {
		t.Errorf("expected DOWN target to have up=0, got %q", output)
	}
	if !strings.Contains(output, `surveiller_target_up{target="unknown_target",address="4.4.4.4",group=""} 0`) {
		t.Errorf("expected UNKNOWN target to have up=0, got %q", output)
	}

	// Check RTT metrics are only present for targets with LastRTT > 0
	if !strings.Contains(output, `surveiller_target_rtt_ms{target="ok_target",address="1.1.1.1",group="group1"} 10`) {
		t.Errorf("expected RTT metric for OK target, got %q", output)
	}
	if !strings.Contains(output, `surveiller_target_rtt_ms{target="warn_target",address="2.2.2.2",group="group1"} 50`) {
		t.Errorf("expected RTT metric for WARN target, got %q", output)
	}

	// Check that targets with 0 RTT don't have RTT metrics
	if strings.Contains(output, `surveiller_target_rtt_ms{target="down_target"`) {
		t.Errorf("should not have RTT metric for DOWN target with 0 RTT, got %q", output)
	}
	if strings.Contains(output, `surveiller_target_rtt_ms{target="unknown_target"`) {
		t.Errorf("should not have RTT metric for UNKNOWN target with 0 RTT, got %q", output)
	}
}

// Test aggregated metrics with all status types
func TestWriteAggregatedAllStatuses(t *testing.T) {
	snapshot := []state.TargetStatus{
		{Status: state.StatusOK},
		{Status: state.StatusOK},
		{Status: state.StatusWarn},
		{Status: state.StatusWarn},
		{Status: state.StatusWarn},
		{Status: state.StatusDown},
		{Status: state.StatusUnknown},
		{Status: state.StatusUnknown},
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	writeAggregated(writer, snapshot)
	_ = writer.Flush()

	got := buf.String()
	expected := strings.Join([]string{
		"surveiller_targets_total 8",
		"surveiller_targets_ok 2",
		"surveiller_targets_warn 3",
		"surveiller_targets_down 1",
		"surveiller_targets_unknown 2",
		"",
	}, "\n")

	if got != expected {
		t.Errorf("unexpected aggregated metrics:\ngot:\n%s\nexpected:\n%s", got, expected)
	}
}

// Test empty snapshot
func TestWriteMetricsEmptySnapshot(t *testing.T) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// Test aggregated with empty snapshot
	writeAggregated(writer, []state.TargetStatus{})
	_ = writer.Flush()

	expected := strings.Join([]string{
		"surveiller_targets_total 0",
		"surveiller_targets_ok 0",
		"surveiller_targets_warn 0",
		"surveiller_targets_down 0",
		"surveiller_targets_unknown 0",
		"",
	}, "\n")

	if buf.String() != expected {
		t.Errorf("unexpected empty aggregated metrics:\n%s", buf.String())
	}

	// Test per-target with empty snapshot
	buf.Reset()
	writer = bufio.NewWriter(&buf)
	writePerTarget(writer, []state.TargetStatus{})
	_ = writer.Flush()

	if buf.String() != "" {
		t.Errorf("expected empty per-target metrics, got %q", buf.String())
	}
}

// Test HTTP server startup and shutdown
func TestServeStartupShutdown(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use port 0 to get a random available port
	err := Serve(ctx, "127.0.0.1:0", config.MetricsModeAggregated, store)

	// Should return context.Canceled when context is cancelled
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("expected context cancellation or deadline exceeded, got %v", err)
	}
}

// Test HTTP server with invalid address
func TestServeInvalidAddress(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use an invalid address format
	err := Serve(ctx, "invalid-address", config.MetricsModeAggregated, store)

	// Should return an error (not context cancellation)
	if err == nil {
		t.Fatalf("expected error for invalid address")
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		t.Fatalf("expected address error, got context error: %v", err)
	}
}

// Test metrics endpoint response headers
func TestHandlerResponseHeaders(t *testing.T) {
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

	expectedContentType := "text/plain; version=0.0.4"
	if contentType := rec.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Fatalf("expected content type %q, got %q", expectedContentType, contentType)
	}
}

// Test HTTP methods other than GET
func TestHandlerUnsupportedMethods(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}
	server := NewServer(config.MetricsModeAggregated, store)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/metrics", nil)
			rec := httptest.NewRecorder()

			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected status 405 for %s method, got %d", method, rec.Code)
			}
		})
	}
}

// Test server error handling during startup
func TestServePortAlreadyInUse(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Try to start server on an invalid port to trigger an error
	err := Serve(ctx, "127.0.0.1:99999", config.MetricsModeAggregated, store)

	// Should return some kind of error (either bind error or context timeout)
	if err == nil {
		t.Fatalf("expected error when starting server on invalid port")
	}
}

// Test metrics endpoint with different paths
func TestHandlerDifferentPaths(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}

	// Create a full mux like in the Serve function
	mux := http.NewServeMux()
	mux.Handle("/metrics", NewServer(config.MetricsModeAggregated, store).Handler())

	tests := []struct {
		path           string
		expectedStatus int
	}{
		{"/metrics", http.StatusOK},
		{"/", http.StatusNotFound},
		{"/health", http.StatusNotFound},
		{"/metrics/extra", http.StatusNotFound}, // Should not match
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d for path %s, got %d", tt.expectedStatus, tt.path, rec.Code)
			}
		})
	}
}

// Test server graceful shutdown
func TestServeGracefulShutdown(t *testing.T) {
	store := fakeStore{
		snapshot: []state.TargetStatus{{Status: state.StatusOK}},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(ctx, "127.0.0.1:0", config.MetricsModeAggregated, store)
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for server to shutdown
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("server did not shutdown within timeout")
	}
}
