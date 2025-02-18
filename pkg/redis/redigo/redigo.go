package redigo

import (
	"context"
	"strings"
	"time"

	rwlockredis "github.com/aldogint/redis-rwlock/pkg/redis"
	"github.com/gomodule/redigo/redis"
)

type pool struct {
	delegate *redis.Pool
}

func (p *pool) Get(ctx context.Context) (rwlockredis.Conn, error) {
	if ctx != nil {
		c, err := p.delegate.GetContext(ctx)
		if err != nil {
			return nil, err
		}
		return &conn{c}, nil
	}
	return &conn{p.delegate.Get()}, nil
}

// NewPool returns a Redigo-based pool implementation.
func NewPool(delegate *redis.Pool) rwlockredis.Pool {
	return &pool{delegate}
}

type conn struct {
	delegate redis.Conn
}

func (c *conn) Get(name string) (string, error) {
	value, err := redis.String(c.delegate.Do("GET", name))
	return value, noErrNil(err)
}

func (c *conn) Set(name string, value string) (bool, error) {
	reply, err := redis.String(c.delegate.Do("SET", name, value))
	return reply == "OK", noErrNil(err)
}

func (c *conn) SetNX(name string, value string, expiry time.Duration) (bool, error) {
	reply, err := redis.String(c.delegate.Do("SET", name, value, "NX", "PX", int(expiry/time.Millisecond)))
	return reply == "OK", noErrNil(err)
}

func (c *conn) PTTL(name string) (time.Duration, error) {
	expiry, err := redis.Int64(c.delegate.Do("PTTL", name))
	return time.Duration(expiry) * time.Millisecond, noErrNil(err)
}

func (c *conn) Eval(script *rwlockredis.Script, keysAndArgs ...interface{}) (interface{}, error) {
	v, err := c.delegate.Do("EVALSHA", args(script, script.Hash, keysAndArgs)...)
	if e, ok := err.(redis.Error); ok && strings.HasPrefix(string(e), "NOSCRIPT ") {
		v, err = c.delegate.Do("EVAL", args(script, script.Src, keysAndArgs)...)
	}
	return v, noErrNil(err)
}

func (c *conn) Close() error {
	err := c.delegate.Close()
	return noErrNil(err)
}

func noErrNil(err error) error {
	if err != redis.ErrNil {
		return err
	}
	return nil
}

func args(script *rwlockredis.Script, spec string, keysAndArgs []interface{}) []interface{} {
	args := make([]interface{}, 1+len(keysAndArgs))
	args[0] = spec
	copy(args[1:], keysAndArgs)
	return args
}
