package snowflake

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// WorkerIdGenerator defines an interface for generating worker ID
type WorkerIdGenerator interface {
	Id() (int64, error)
}

// Snowflake algorithm constants
const (
	epoch          = int64(1672502400000)
	timestampBits  = uint(41)
	workeridBits   = uint(16)
	sequenceBits   = uint(6)
	timestampMax   = int64(-1 ^ (-1 << timestampBits))
	workeridMax    = int64(-1 ^ (-1 << workeridBits))
	sequenceMask   = int64(-1 ^ (-1 << sequenceBits))
	workeridShift  = sequenceBits
	timestampShift = sequenceBits + workeridBits
)

// Snowflake represents a snowflake ID generator
type Snowflake struct {
	sync.Mutex
	timestamp         int64
	workerIdGenerator WorkerIdGenerator
	sequence          int64
}

// NewSnowFlake creates a new Snowflake instance with optional worker ID generator
// If workerIdGenerator is nil, uses local IP based generator by default
func NewSnowFlake(workerIdGenerator WorkerIdGenerator) *Snowflake {
	if workerIdGenerator == nil {
		workerIdGenerator = &localIPWorkerIdGenerator{localIPv4}
	}

	return &Snowflake{
		workerIdGenerator: workerIdGenerator,
	}
}

// NextVal generates the next unique ID using the snowflake algorithm
func (s *Snowflake) NextVal() (int64, error) {
	workerid, err := s.getWorkerId()
	if err != nil {
		return 0, err
	}

	s.Lock()
	defer s.Unlock()

	if s.workerIdGenerator == nil {
		return 0, fmt.Errorf("worker id generator is nil")
	}

	now := time.Now().UnixNano() / 1000000
	if s.timestamp == now {
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			now = s.waitNextMillis(s.timestamp)
		}
	} else {
		s.sequence = 0
	}

	t := now - epoch
	if t > timestampMax {
		return 0, fmt.Errorf("epoch must be between 0 and %d", timestampMax-1)
	}

	s.timestamp = now
	r := int64((t)<<timestampShift | (workerid << workeridShift) | (s.sequence))
	return r, nil
}

func (s *Snowflake) getWorkerId() (int64, error) {
	if s.workerIdGenerator == nil {
		return 0, fmt.Errorf("worker id generator is nil")
	}

	workerid, err := s.workerIdGenerator.Id()
	if err != nil {
		return 0, err
	}

	if workerid > workeridMax || workerid < 0 {
		return 0, fmt.Errorf("worker id can't be greater than %d or less than 0", workerid)
	}
	return workerid, nil
}

func (s *Snowflake) waitNextMillis(lastTimestamp int64) int64 {
	now := time.Now().UnixNano() / 1000000
	for now <= lastTimestamp {
		now = time.Now().UnixNano() / 1000000
	}
	return now
}

type localIPWorkerIdGenerator struct {
	localIP func() (net.IP, error)
}

func (l *localIPWorkerIdGenerator) Id() (int64, error) {
	ip, err := l.localIP()
	if err != nil {
		return 0, err
	}
	return int64(ip[2])<<8 + int64(ip[3]), nil
}

func localIPv4() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var ipV4s []net.IP
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.IsLinkLocalMulticast() || ipnet.IP.IsLinkLocalUnicast() {
			continue
		}

		ipV4s = append(ipV4s, ipnet.IP.To4())
	}

	if len(ipV4s) == 0 {
		return nil, fmt.Errorf("no valid ipv4 address")
	}
	return ipV4s[rand.Intn(len(ipV4s))], nil
}
