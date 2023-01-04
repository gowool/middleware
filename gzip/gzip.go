package gzip

import (
	"compress/gzip"
	"github.com/NYTimes/gziphandler"
	"github.com/gowool/wool"
	"net/http"
)

type Config struct {
	Level int `mapstructure:"level"`

	// MinSize is the minimum size until we enable gzip compression.
	// 1500 bytes is the MTU size for the internet since that is the largest size allowed at the network layer.
	// If you take a file that is 1300 bytes and compress it to 800 bytes, it’s still transmitted in that same 1500 byte packet regardless, so you’ve gained nothing.
	// That being the case, you should restrict the gzip compression to files with a size greater than a single packet, 1400 bytes (1.4KB) is a safe value.
	MinSize int `mapstructure:"min_size"`
}

func (cfg *Config) init() {
	if cfg.Level == gzip.NoCompression {
		cfg.Level = gzip.DefaultCompression
	}
	if cfg.MinSize == 0 {
		cfg.MinSize = gziphandler.DefaultMinSize
	}
}

type Gzip struct {
	wrapper func(http.Handler) http.Handler
}

func Middleware(cfg Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg Config) *Gzip {
	cfg.init()

	wrapper, err := gziphandler.NewGzipLevelAndMinSize(cfg.Level, cfg.MinSize)
	if err != nil {
		panic(err)
	}

	return &Gzip{wrapper: wrapper}
}

func (m *Gzip) Middleware(next wool.Handler) wool.Handler {
	return wool.ToMiddleware(m.wrapper)(next)
}
