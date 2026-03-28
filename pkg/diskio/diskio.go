package diskio

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

// Mode selects the disk IO access pattern.
type Mode string

const (
	// ModeTxnRW writes a batch, fsyncs, then reads a different random batch.
	// Simulates transactional database IO.
	ModeTxnRW Mode = "txn_rw"
	// ModeSequentialRW writes and reads batches sequentially through the file.
	ModeSequentialRW Mode = "sequential_rw"
	// ModeRandomizedRW writes and reads batches at independent random positions.
	ModeRandomizedRW Mode = "randomized_rw"
)

// ParseMode validates a mode string and returns the corresponding Mode.
func ParseMode(s string) (Mode, error) {
	switch Mode(s) {
	case ModeTxnRW, ModeSequentialRW, ModeRandomizedRW:
		return Mode(s), nil
	default:
		return "", fmt.Errorf("unknown io_mode %q: valid values are txn_rw, sequential_rw, randomized_rw", s)
	}
}

// Config holds disk IO scenario configuration.
type Config struct {
	Mode        Mode
	FilePath    string
	BatchSizeKB int
	FileSizeMB  int
}

// Scenario implements bench.Scenario for disk IO load testing.
type Scenario struct {
	config     Config
	batchSize  int64
	fileSize   int64
	numBatches int64

	initOnce sync.Once
	initErr  error
	file     *os.File
	rng      *rand.Rand
	writeBuf []byte
	readBuf  []byte
	seqPos   int64 // next sequential batch index
}

// New creates a new disk IO Scenario.
func New(config Config) *Scenario {
	batchSize := int64(config.BatchSizeKB) * 1024
	fileSize := int64(config.FileSizeMB) * 1024 * 1024
	return &Scenario{
		config:     config,
		batchSize:  batchSize,
		fileSize:   fileSize,
		numBatches: fileSize / batchSize,
	}
}

// Name implements bench.Scenario.
func (s *Scenario) Name() string { return "disk" }

// Run performs one write+read IO operation in the configured mode.
// The data file is lazily created and pre-populated on the first call.
func (s *Scenario) Run(_ context.Context) bench.Result {
	if err := s.ensureInit(); err != nil {
		return bench.Result{Err: err}
	}

	start := time.Now()
	var err error

	switch s.config.Mode {
	case ModeTxnRW:
		err = s.runTxnRW()
	case ModeSequentialRW:
		err = s.runSequentialRW()
	case ModeRandomizedRW:
		err = s.runRandomizedRW()
	}

	return bench.Result{Duration: time.Since(start), Err: err}
}

// Close releases the data file. Safe to call more than once.
func (s *Scenario) Close() error {
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// --- initialisation ---------------------------------------------------------

func (s *Scenario) ensureInit() error {
	s.initOnce.Do(func() {
		s.initErr = s.open()
	})
	return s.initErr
}

func (s *Scenario) open() error {
	f, err := os.OpenFile(s.config.FilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	s.file = f

	s.rng = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
	s.writeBuf = make([]byte, s.batchSize)
	s.readBuf = make([]byte, s.batchSize)

	// Fill the write buffer once; content is irrelevant for IO benchmarking.
	s.fillRandom(s.writeBuf)

	// Pre-populate the file to target size in 1 MB chunks so that
	// subsequent reads hit real allocated blocks rather than sparse holes.
	const chunkSize = 1 << 20 // 1 MB
	chunk := make([]byte, chunkSize)
	s.fillRandom(chunk)

	for pos := int64(0); pos < s.fileSize; pos += chunkSize {
		n := int64(chunkSize)
		if pos+n > s.fileSize {
			n = s.fileSize - pos
		}
		if _, err := f.WriteAt(chunk[:n], pos); err != nil {
			return err
		}
	}
	return f.Sync()
}

func (s *Scenario) fillRandom(buf []byte) {
	for i := 0; i+8 <= len(buf); i += 8 {
		binary.LittleEndian.PutUint64(buf[i:], s.rng.Uint64())
	}
	// Handle trailing bytes (<8) if batch size is not a multiple of 8.
	tail := len(buf) % 8
	if tail > 0 {
		v := s.rng.Uint64()
		for j := range tail {
			buf[len(buf)-tail+j] = byte(v >> (j * 8))
		}
	}
}

func (s *Scenario) randomOffset() int64 {
	return s.rng.Int64N(s.numBatches) * s.batchSize
}

// --- IO modes ---------------------------------------------------------------

// runTxnRW writes a batch, fsyncs, then reads a different random batch.
func (s *Scenario) runTxnRW() error {
	writeOff := s.randomOffset()
	s.fillRandom(s.writeBuf)
	if _, err := s.file.WriteAt(s.writeBuf, writeOff); err != nil {
		return err
	}
	if err := s.file.Sync(); err != nil {
		return err
	}

	readOff := s.randomOffset()
	for readOff == writeOff && s.numBatches > 1 {
		readOff = s.randomOffset()
	}
	_, err := s.file.ReadAt(s.readBuf, readOff)
	return err
}

// runSequentialRW writes then reads the next batch in sequence, wrapping at EOF.
func (s *Scenario) runSequentialRW() error {
	off := (s.seqPos % s.numBatches) * s.batchSize
	s.seqPos++

	s.fillRandom(s.writeBuf)
	if _, err := s.file.WriteAt(s.writeBuf, off); err != nil {
		return err
	}
	_, err := s.file.ReadAt(s.readBuf, off)
	return err
}

// runRandomizedRW writes and reads at independent random positions.
func (s *Scenario) runRandomizedRW() error {
	s.fillRandom(s.writeBuf)
	if _, err := s.file.WriteAt(s.writeBuf, s.randomOffset()); err != nil {
		return err
	}
	_, err := s.file.ReadAt(s.readBuf, s.randomOffset())
	return err
}
