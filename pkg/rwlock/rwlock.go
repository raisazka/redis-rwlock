package rwlock

import (
	"errors"
	"strconv"
	"time"

	"github.com/aldogint/redis-rwlock/pkg/redis"
	"github.com/gofrs/uuid"
)

var (
	// ErrConnection is returned by Locker methods in case of problems with redis.
	ErrConnection = errors.New("redis connection error")
	// ErrTimeout is returned by Locker methods if timeout was specified and was exceeded while waiting for lock.
	ErrTimeout = errors.New("timeout exceeded but lock not acquired")
	// ErrInterrupted is returned by Locker methods if they were interrupted via Context.
	ErrInterrupted = errors.New("interrupted")
	// ErrNotReleased is returned by locker methods if lock was not released.
	ErrNotReleased = errors.New("lock was not released")
	// ErrUnknownMode is return by locker methods in case the lock was set to unknown mode.
	ErrUnknownMode = errors.New("lock is in unknown mode")
)

// Locker allows to execute given functions at reader or writer access privilege.
type Locker interface {
	// Read executes given function with shared reader access.
	Read(fn func()) error
	// Write executes given function with unique writer access.
	Write(fn func()) error
}

// New instance of RW-Locker.
// keyLock, keyReadersCount, keyWriterIntent must be unique keys that will be used by locker implementation.
func New(pool redis.Pool, keyLock, keyReadersCount, keyWriterIntent string, opts Options) Locker {
	prepareOpts(&opts)
	return &lockerImpl{
		redisPool:       pool,
		options:         opts,
		keyGlobalLock:   keyLock,
		keyReadersCount: keyReadersCount,
		keyWriterIntent: keyWriterIntent,
		writerToken:     makeToken(opts.AppID),
		lockTTL:         strconv.FormatInt(int64(opts.LockTTL/time.Millisecond), 10),
	}
}

func makeToken(prefix string) string {
	token := uuid.Must(uuid.NewV4()).String()
	if len(prefix) > 0 {
		token = prefix + "_" + token
	}
	return token
}
