package connection

import (
	"sync/atomic"
)

// SeqGenerator is a lock-free, thread-safe sequence number generator
// for 16-bit protocol frame sequence counters (0..65535).
type SeqGenerator struct {
	cur atomic.Uint32
}

// NewSeqGenerator creates a new SeqGenerator initialized so that
// the first call to Next() returns 0.
func NewSeqGenerator() *SeqGenerator {
	s := &SeqGenerator{}
	// Set to 0xFFFFFFFF so that atomic.Add(1) overflows to 0.
	s.cur.Store(^uint32(0))
	return s
}

// NewSeqGeneratorWithStart creates a SeqGenerator starting at a specific value,
// such that the next call to Next() returns start.
func NewSeqGeneratorWithStart(start uint16) *SeqGenerator {
	s := &SeqGenerator{}
	s.cur.Store(uint32(start) - 1)
	return s
}

// Next atomically generates the next uint16 sequence number.
// When reaching 65535 (0xFFFF), it wraps cleanly to 0 (0x0000).
func (s *SeqGenerator) Next() uint16 {
	return uint16(s.cur.Add(1))
}

// Current returns the current sequence number without incrementing.
func (s *SeqGenerator) Current() uint16 {
	return uint16(s.cur.Load())
}

// Reset resets the generator so that the subsequent call to Next()
// will return val.
func (s *SeqGenerator) Reset(val uint16) {
	s.cur.Store(uint32(val) - 1)
}
