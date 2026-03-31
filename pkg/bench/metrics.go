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

	// Baseline mode only — omitted in benchmark mode or when no threshold is set.
	BaselineMet    *int     `json:"baseline_met,omitempty"`
	TargetReadBPS  *int64   `json:"target_read_bps,omitempty"`
	TargetWriteBPS *int64   `json:"target_write_bps,omitempty"`
	TargetOpsPerSec *float64 `json:"target_ops_per_sec,omitempty"`
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
	TargetReadBPS   int64
	TargetWriteBPS  int64
	TargetOpsPerSec float64
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

	// Emit threshold values and baseline_met only when a threshold is configured.
	hasThreshold := in.TargetReadBPS > 0 || in.TargetWriteBPS > 0 || in.TargetOpsPerSec > 0
	if in.Mode == ModeBaseline && hasThreshold {
		met := computeBaselineMet(m.OpsPerSec, m.BytesReadPerSec, m.BytesWrittenPerSec,
			in.TargetOpsPerSec, in.TargetReadBPS, in.TargetWriteBPS)
		m.BaselineMet = &met
		if in.TargetReadBPS > 0 {
			v := in.TargetReadBPS
			m.TargetReadBPS = &v
		}
		if in.TargetWriteBPS > 0 {
			v := in.TargetWriteBPS
			m.TargetWriteBPS = &v
		}
		if in.TargetOpsPerSec > 0 {
			v := in.TargetOpsPerSec
			m.TargetOpsPerSec = &v
		}
	}

	return m
}

// computeBaselineMet returns 1 if all configured targets are reached at or
// above 98%, 0 otherwise.  A target of 0 is always considered met.
func computeBaselineMet(actualOPS, actualReadBPS, actualWriteBPS float64, targetOPS float64, targetReadBPS, targetWriteBPS int64) int {
	const threshold = 0.98
	if targetOPS > 0 && actualOPS < threshold*targetOPS {
		return 0
	}
	if targetReadBPS > 0 && actualReadBPS < threshold*float64(targetReadBPS) {
		return 0
	}
	if targetWriteBPS > 0 && actualWriteBPS < threshold*float64(targetWriteBPS) {
		return 0
	}
	return 1
}
