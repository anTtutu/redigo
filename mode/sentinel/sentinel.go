package sentinel

import (
	"runtime"
	"time"

	"github.com/FZambia/sentinel"
	"github.com/gomodule/redigo/redis"
	"github.com/letsfire/redigo/mode"
)

type sentinelMode struct {
	pool *redis.Pool
}

func (sm *sentinelMode) GetConn() redis.Conn {
	return sm.pool.Get()
}

func (sm *sentinelMode) NewConn() (redis.Conn, error) {
	return sm.pool.Dial()
}

func (sm *sentinelMode) String() string {
	return "sentinel"
}

var _ mode.IMode = &sentinelMode{}

func New(optFuncs ...OptFunc) *sentinelMode {
	opts := options{
		addrs:      []string{"127.0.0.1:26379"},
		masterName: "mymaster",
		poolOpts:   mode.DefaultPoolOpts(),
		dialOpts:   mode.DefaultDialOpts(),
	}
	for _, optFunc := range optFuncs {
		optFunc(&opts)
	}
	if len(opts.sentinelDialOpts) == 0 {
		opts.sentinelDialOpts = opts.dialOpts
	}
	st := &sentinel.Sentinel{
		Addrs:      opts.addrs,
		MasterName: opts.masterName,
		Pool: func(addr string) *redis.Pool {
			stp := &redis.Pool{
				Wait:    true,
				MaxIdle: runtime.GOMAXPROCS(0),
				Dial: func() (redis.Conn, error) {
					return redis.Dial("tcp", addr, opts.sentinelDialOpts...)
				},
				TestOnBorrow: func(c redis.Conn, t time.Time) (err error) {
					_, err = c.Do("PING")
					return
				},
			}
			return stp
		},
	}
	pool := &redis.Pool{
		Dial: func() (conn redis.Conn, err error) {
			addr, err := st.MasterAddr()
			if err != nil {
				return
			}
			return redis.Dial("tcp", addr, opts.dialOpts...)
		},
	}
	for _, poolOptFunc := range opts.poolOpts {
		poolOptFunc(pool)
	}
	return &sentinelMode{pool: pool}
}
