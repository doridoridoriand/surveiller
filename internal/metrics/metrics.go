package metrics

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/doridoridoriand/deadman-go/internal/config"
	"github.com/doridoridoriand/deadman-go/internal/state"
)

// Server exposes Prometheus-style metrics based on current state.
type Server struct {
	mode  config.MetricsMode
	store state.Store
}

// NewServer constructs a metrics server.
func NewServer(mode config.MetricsMode, store state.Store) *Server {
	return &Server{mode: mode, store: store}
}

// Handler returns an http handler that serves metrics.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		bw := bufio.NewWriter(w)
		defer bw.Flush()
		s.writeMetrics(bw)
	})
}

func (s *Server) writeMetrics(w *bufio.Writer) {
	snapshot := s.store.GetSnapshot()
	if s.mode == "" {
		return
	}

	if s.mode == config.MetricsModeAggregated || s.mode == config.MetricsModeBoth {
		writeAggregated(w, snapshot)
	}
	if s.mode == config.MetricsModePerTarget || s.mode == config.MetricsModeBoth {
		writePerTarget(w, snapshot)
	}
}

func writeAggregated(w *bufio.Writer, snapshot []state.TargetStatus) {
	total := len(snapshot)
	var okCount, warnCount, downCount, unknownCount int
	for _, target := range snapshot {
		switch target.Status {
		case state.StatusOK:
			okCount++
		case state.StatusWarn:
			warnCount++
		case state.StatusDown:
			downCount++
		default:
			unknownCount++
		}
	}
	fmt.Fprintf(w, "deadman_targets_total %d\n", total)
	fmt.Fprintf(w, "deadman_targets_ok %d\n", okCount)
	fmt.Fprintf(w, "deadman_targets_warn %d\n", warnCount)
	fmt.Fprintf(w, "deadman_targets_down %d\n", downCount)
	fmt.Fprintf(w, "deadman_targets_unknown %d\n", unknownCount)
}

func writePerTarget(w *bufio.Writer, snapshot []state.TargetStatus) {
	for _, target := range snapshot {
		labels := fmt.Sprintf(
			"target=%q,address=%q,group=%q",
			escapeLabel(target.Name),
			escapeLabel(target.Address),
			escapeLabel(target.Group),
		)
		up := 0
		if target.Status == state.StatusOK {
			up = 1
		}
		fmt.Fprintf(w, "deadman_target_up{%s} %d\n", labels, up)
		if target.LastRTT > 0 {
			fmt.Fprintf(w, "deadman_target_rtt_ms{%s} %d\n", labels, target.LastRTT.Milliseconds())
		}
	}
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return value
}

// Serve starts an HTTP server and blocks until context cancellation.
func Serve(ctx context.Context, addr string, mode config.MetricsMode, store state.Store) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", NewServer(mode, store).Handler())
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return context.Canceled
		}
		return err
	}
}
