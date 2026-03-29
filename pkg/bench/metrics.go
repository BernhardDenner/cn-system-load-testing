package bench

import "time"

// Mode values for Metrics.
const (
	ModeBenchmark = "benchmark"
	ModeBaseline  = "baseline"
)

// Phase values for Metrics.
const (
	PhaseRunning = "running"
	PhaseSummary = "summary"
)

// Metrics is the JSON record emitted to stdout at each reporting interval
// and as a final summary at the end of a run.
type Metrics struct {
	Timestamp string  `json:"timestamp"`
	Mode      string  `json:"mode"`
	Phase     string  `json:"phase"`
	Module    string  `json:"module"`
	Ops       int64   `json:"ops"`
	OpsPerSec float64 `json:"ops_per_sec"`
	AvgLatMs  float64 `json:"avg_latency_ms,omitempty"`
	Errors    int64   `json:"errors"`

	// Disk IO specific — omitted when zero.
	BytesRead          int64   `json:"bytes_read,omitempty"`
	BytesWritten       int64   `json:"bytes_written,omitempty"`
	BytesReadPerSec    float64 `json:"bytes_read_per_sec,omitempty"`
	BytesWrittenPerSec float64 `json:"bytes_written_per_sec,omitempty"`

	// Baseline mode only — omitted in benchmark mode.
	BaselineMet *int `json:"baseline_met,omitempty"`
}

// MetricsInput provides the raw counters needed to build a Metrics record.
// Total* fields are cumulative over the entire run.
// Interval* fields cover the period since the previous measurement and are
// used to compute per-second rates and average latency.
type MetricsInput struct {
	Mode    string
	Module  string
	Phase   string
	Elapsed time.Duration // time since previous measurement

	TotalOps    int64
	TotalErrors int64

	IntervalOps       int64
	IntervalLatencyNs int64

	TotalBytesRead       int64
	TotalBytesWritten    int64
	IntervalBytesRead    int64
	IntervalBytesWritten int64

	// Baseline targets — 0 means no target for that direction.
	TargetReadBPS  int64
	TargetWriteBPS int64
}

// NewMetrics constructs a Metrics record from the given input.
func NewMetrics(in MetricsInput) Metrics {
	m := Metrics{
		Timestamp:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		Mode:         in.Mode,
		Phase:        in.Phase,
		Module:       in.Module,
		Ops:          in.TotalOps,
		Errors:       in.TotalErrors,
		BytesRead:    in.TotalBytesRead,
		BytesWritten: in.TotalBytesWritten,
	}
	if secs := in.Elapsed.Seconds(); secs > 0 {
		m.OpsPerSec = float64(in.IntervalOps) / secs
		m.BytesReadPerSec = float64(in.IntervalBytesRead) / secs
		m.BytesWrittenPerSec = float64(in.IntervalBytesWritten) / secs
	}
	if in.IntervalOps > 0 {
		m.AvgLatMs = float64(in.IntervalLatencyNs) / float64(in.IntervalOps) / 1e6
	}

	// Compute baseline_met for baseline mode.
	if in.Mode == ModeBaseline {
		met := computeBaselineMet(m.BytesReadPerSec, m.BytesWrittenPerSec, in.TargetReadBPS, in.TargetWriteBPS)
		m.BaselineMet = &met
	}

	return m
}

// computeBaselineMet returns 1 if the actual rates reach at least 98% of
// the configured targets, 0 otherwise.  A target of 0 is always considered met.
func computeBaselineMet(actualReadBPS, actualWriteBPS float64, targetReadBPS, targetWriteBPS int64) int {
	const threshold = 0.98
	if targetReadBPS > 0 && actualReadBPS < threshold*float64(targetReadBPS) {
		return 0
	}
	if targetWriteBPS > 0 && actualWriteBPS < threshold*float64(targetWriteBPS) {
		return 0
	}
	return 1
}
