package unixid

import (
	"sync"
	"webtyp.com/time"

	. "webtyp.com/fmt"
)

var now = time.Now

// Replica identifies one device that mints ids while offline. The server assigns it
// (never this library). 0 is not a valid replica: it means "minted by a server".
type Replica uint32

type idError string

func (e idError) Error() string { return string(e) }

const ErrReplicaZero idError = "unixid: replica 0 is reserved for servers"

// UnixID is the main struct for ID generation and handling
type UnixID struct {
	mu     sync.Mutex
	last   int64
	suffix string
}

// NewUnixID returns the server generator: ids are "<nanoseconds>" with no suffix.
// The error is always nil today; it is kept so both constructors share one shape.
func NewUnixID() (*UnixID, error) {
	return &UnixID{
		suffix: "",
	}, nil
}

// NewForReplica returns a generator for one device: ids are "<nanoseconds>.<replica>".
// last is the timestamp part of the newest id this replica already minted (0 on a fresh
// device); every id this generator returns has a timestamp strictly greater than last.
func NewForReplica(replica Replica, last int64) (*UnixID, error) {
	if replica == 0 {
		return nil, ErrReplicaZero
	}
	return &UnixID{
		last:   last,
		suffix: "." + Convert(uint32(replica)).String(),
	}, nil
}

// NewID generates a new unique ID based on Unix nanosecond timestamp (UTC).
func (id *UnixID) NewID() string {
	id.mu.Lock()
	defer id.mu.Unlock()

	current := now()
	if current <= id.last {
		current = id.last + 1
	}
	id.last = current

	return Convert(current).String() + id.suffix
}
