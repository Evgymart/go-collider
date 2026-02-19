package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	NodeIDBits    = 10
	SequenceBits  = 12
	TimestampBits = 41

	MaxNodeID    = -1 ^ (-1 << NodeIDBits)
	MaxSequence  = -1 ^ (-1 << SequenceBits)
	MaxTimestamp = int64(1<<TimestampBits - 1)

	NodeIDShift    = SequenceBits
	TimestampShift = SequenceBits + NodeIDBits

	DefaultEpoch = int64(1704067200000)
)

// Snowflake is a distributed unique ID generator inspired by Twitter's Snowflake.
// Each ID is 64 bits long and consists of:
// - 41 bits for timestamp (milliseconds since epoch)
// - 10 bits for node ID (supports up to 1024 nodes)
// - 12 bits for sequence number (supports up to 4096 IDs per millisecond per node)
type Snowflake struct {
	mu       sync.Mutex
	lastTime int64
	nodeID   int64
	sequence int64
	epoch    int64
}

var (
	ErrInvalidNodeID     = errors.New("node ID must be between 0 and 1023")
	ErrBackwardsTime     = errors.New("system clock moved backwards")
	ErrTimestampOverflow = errors.New("timestamp overflow: snowflake ID exceeds 41-bit timestamp capacity")
)

func New(nodeID int64) (*Snowflake, error) {
	return NewWithEpoch(nodeID, DefaultEpoch)
}

func NewWithEpoch(nodeID int64, epoch int64) (*Snowflake, error) {
	if nodeID < 0 || nodeID > MaxNodeID {
		return nil, ErrInvalidNodeID
	}

	return &Snowflake{
		nodeID:   nodeID,
		epoch:    epoch,
		lastTime: 0,
		sequence: 0,
	}, nil
}

func (s *Snowflake) Generate() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < s.lastTime {
		return 0, ErrBackwardsTime
	}

	if now == s.lastTime {
		s.sequence = (s.sequence + 1) & MaxSequence
		if s.sequence == 0 {
			for now <= s.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}

	s.lastTime = now

	timeSinceEpoch := now - s.epoch

	if timeSinceEpoch > MaxTimestamp {
		return 0, ErrTimestampOverflow
	}

	id := (timeSinceEpoch << TimestampShift) |
		(s.nodeID << NodeIDShift) |
		s.sequence

	return id, nil
}

func (s *Snowflake) MustGenerate() int64 {
	id, err := s.Generate()
	if err != nil {
		panic(err)
	}
	return id
}

func Parse(id int64) (timestamp int64, nodeID int64, sequence int64) {
	timestamp = (id >> TimestampShift) + DefaultEpoch
	nodeID = (id >> NodeIDShift) & MaxNodeID
	sequence = id & MaxSequence
	return
}

func ParseTime(id int64) time.Time {
	millis := (id >> TimestampShift) + DefaultEpoch
	return time.UnixMilli(millis)
}

func ParseWithEpoch(id int64, epoch int64) (timestamp int64, nodeID int64, sequence int64) {
	timestamp = (id >> TimestampShift) + epoch
	nodeID = (id >> NodeIDShift) & MaxNodeID
	sequence = id & MaxSequence
	return
}

func ParseTimeWithEpoch(id int64, epoch int64) time.Time {
	millis := (id >> TimestampShift) + epoch
	return time.UnixMilli(millis)
}

func (s *Snowflake) Epoch() int64 {
	return s.epoch
}
