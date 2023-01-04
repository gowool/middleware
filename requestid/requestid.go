package requestid

import (
	"github.com/google/uuid"
	"github.com/gowool/wool"
)

type Config struct {
	TargetHeader     string `mapstructure:"target_header"`
	RequestIDHandler func(c wool.Ctx, requestID string) error
}

func (cfg *Config) init() {
	if cfg.TargetHeader == "" {
		cfg.TargetHeader = wool.HeaderXRequestID
	}
}

type RequestID struct {
	cfg Config
}

func Middleware(cfg Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg Config) *RequestID {
	cfg.init()

	return &RequestID{cfg: cfg}
}

func (m *RequestID) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		if m.cfg.TargetHeader != "" {
			rid := c.Req().Header.Get(m.cfg.TargetHeader)
			if rid == "" {
				rid = uuid.NewString()
			}
			c.Res().Header().Set(m.cfg.TargetHeader, rid)

			if m.cfg.RequestIDHandler != nil {
				if err := m.cfg.RequestIDHandler(c, rid); err != nil {
					return err
				}
			}
		}
		return next(c)
	}
}
