package memory

import (
	"context"
	"encoding/binary"
	"math/rand/v2"
	"time"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

// blockSize is the allocation unit for each operation.
const blockSize int64 = 1 << 20 // 1 MB

// Config holds memory scenario configuration.
type Config struct {
	// MaxUseMB is the maximum heap memory in MB the scenario may occupy.
	// A value of 0 means auto-detect from the cgroup limit or total system RAM.
	MaxUseMB int
}

// Scenario implements bench.Scenario for memory load testing.
// It maintains a ring buffer of 1 MB blocks.  Each Run() allocates a new
// block (write stress), stores it in the ring (evicting the oldest if full),
// and reads a random existing block — exercising allocation, GC, and the
// memory read/write paths.
type Scenario struct {
	config    Config
	maxBytes  int64
	numBlocks int
	pool      [][]byte
	writeIdx  int
	occupied  int // how many slots in pool are non-nil
	rng       *rand.Rand
	readBuf   []byte
}

// New creates a new memory load test Scenario.
func New(config Config) *Scenario {
	maxBytes := int64(config.MaxUseMB) * 1024 * 1024
	if maxBytes <= 0 {
		maxBytes = availableMemoryBytes()
	}
	numBlocks := int(maxBytes / blockSize)
	if numBlocks < 1 {
		numBlocks = 1
	}
	return &Scenario{
		config:    config,
		maxBytes:  maxBytes,
		numBlocks: numBlocks,
		pool:      make([][]byte, numBlocks),
		rng:       rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
		readBuf:   make([]byte, blockSize),
	}
}

// Name implements bench.Scenario.
func (s *Scenario) Name() string { return "memory" }

// Run performs one allocate → write → read → (evict) cycle.
func (s *Scenario) Run(_ context.Context) bench.Result {
	start := time.Now()

	// Allocate a fresh block and fill it with pseudo-random data.
	block := make([]byte, blockSize)
	s.fillRandom(block)

	// Store in the ring buffer, overwriting (and thus freeing) the oldest
	// block when the pool is full.
	s.pool[s.writeIdx] = block
	if s.occupied < s.numBlocks {
		s.occupied++
	}

	// Read from a random occupied slot into readBuf.
	readIdx := s.rng.IntN(s.occupied)
	copy(s.readBuf, s.pool[readIdx])

	s.writeIdx = (s.writeIdx + 1) % s.numBlocks

	return bench.Result{
		Duration:     time.Since(start),
		BytesRead:    blockSize,
		BytesWritten: blockSize,
	}
}

func (s *Scenario) fillRandom(buf []byte) {
	for i := 0; i+8 <= len(buf); i += 8 {
		binary.LittleEndian.PutUint64(buf[i:], s.rng.Uint64())
	}
}
