package favicon

import (
	"github.com/gowool/wool"
	"io"
	"net/http"
	"os"
	"strconv"
)

const (
	fPath  = "/favicon.ico"
	hAllow = "GET,HEAD,OPTIONS"
	hZero  = "0"
)

type Config struct {
	File         string `mapstructure:"file"`
	CacheControl string `mapstructure:"cache_control"`
	FileSystem   http.FileSystem
}

func (cfg *Config) init() {
	if cfg.CacheControl == "" {
		cfg.CacheControl = "public, max-age=31536000"
	}
}

type Favicon struct {
	cfg     Config
	icon    []byte
	iconLen string
}

func Middleware(cfg Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg Config) *Favicon {
	cfg.init()

	m := &Favicon{cfg: cfg}

	var err error
	if cfg.File != "" {
		if cfg.FileSystem != nil {
			f, err := cfg.FileSystem.Open(cfg.File)
			if err != nil {
				panic(err)
			}
			if m.icon, err = io.ReadAll(f); err != nil {
				panic(err)
			}
		} else if m.icon, err = os.ReadFile(cfg.File); err != nil {
			panic(err)
		}

		m.iconLen = strconv.Itoa(len(m.icon))
	}

	return m
}

func (m *Favicon) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		if c.Req().URL.Path != fPath {
			return next(c)
		}

		method := c.Req().Method
		if method != http.MethodGet && method != http.MethodHead {
			c.Res().Header().Set(wool.HeaderContentLength, hZero)
			if method == http.MethodOptions {
				return c.OK()
			}

			c.Res().Header().Set(wool.HeaderAllow, hAllow)
			return c.Status(http.StatusMethodNotAllowed)
		}

		if len(m.icon) > 0 {
			c.Res().Header().Set(wool.HeaderContentLength, m.iconLen)
			c.Res().Header().Set(wool.HeaderCacheControl, m.cfg.CacheControl)
			return c.Blob(http.StatusOK, wool.MIMEImageIcon, m.icon)
		}

		return c.NoContent()
	}
}
