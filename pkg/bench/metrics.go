package bench

import "time"

// Phase values for Metrics.
const (
	PhaseRunning = "running"
	PhaseSummary = "summary"
)

// Metrics is the JSON record emitted to stdout at each reporting interval
// and as a final summary at the end of a run.
type Metrics struct {
	Timestamp string  `json:"timestamp"`
	Phase     string  `json:"phase"`
	Module    string  `json:"module"`
	Ops       int64   `json:"ops"`
	OpsPerSec float64 `json:"ops_per_sec"`
	AvgLatMs  float64 `json:"avg_latency_ms,omitempty"`
	Errors    int64   `json:"errors"`
}

// NewMetrics constructs a Metrics record from raw accumulated counters.
func NewMetrics(module, phase string, ops, errors, latencyNs int64, elapsed time.Duration) Metrics {
	m := Metrics{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Phase:     phase,
		Module:    module,
		Ops:       ops,
		OpsPerSec: float64(ops) / elapsed.Seconds(),
		Errors:    errors,
	}
	if ops > 0 {
		m.AvgLatMs = float64(latencyNs) / float64(ops) / 1e6
	}
	return m
}
