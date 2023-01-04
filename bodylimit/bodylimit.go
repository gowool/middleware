package bodylimit

import (
	"github.com/gowool/wool"
	"io"
	"sync"
)

type Config struct {
	LimitBytes int64 `mapstructure:"limit_bytes"`
}

type BodyLimit struct {
	cfg  Config
	pool sync.Pool
}

func Middleware(cfg Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg Config) *BodyLimit {
	return &BodyLimit{
		cfg: cfg,
		pool: sync.Pool{
			New: func() interface{} {
				return &limitedReader{limitBytes: cfg.LimitBytes}
			},
		},
	}
}

func (m *BodyLimit) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		if c.Req().ContentLength > m.cfg.LimitBytes {
			return wool.NewErrRequestEntityTooLarge(nil)
		}

		r := m.pool.Get().(*limitedReader)
		r.Reset(c.Req().Body)
		defer m.pool.Put(r)
		c.Req().Body = r

		return next(c)
	}
}

type limitedReader struct {
	limitBytes int64
	read       int64
	reader     io.ReadCloser
}

func (r *limitedReader) Read(b []byte) (n int, err error) {
	n, err = r.reader.Read(b)
	r.read += int64(n)
	if r.read > r.limitBytes {
		return n, wool.NewErrRequestEntityTooLarge(nil)
	}
	return
}

func (r *limitedReader) Close() error {
	return r.reader.Close()
}

func (r *limitedReader) Reset(reader io.ReadCloser) {
	r.reader = reader
	r.read = 0
}
