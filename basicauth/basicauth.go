package basicauth

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/gowool/wool"
	"strconv"
	"strings"
)

const basic = "basic"

type Validator func(c wool.Ctx, user, password string) (bool, error)

type Config struct {
	Realm     string `mapstructure:"realm"`
	Validator Validator
}

func (cfg *Config) Init() {
	if cfg.Validator == nil {
		panic(errors.New("basic-auth middleware requires a validator function"))
	}

	if cfg.Realm == "" {
		cfg.Realm = "Restricted"
	} else {
		cfg.Realm = strconv.Quote(cfg.Realm)
	}
}

type BasicAuth struct {
	cfg *Config
}

func Middleware(cfg *Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg *Config) *BasicAuth {
	cfg.Init()

	return &BasicAuth{cfg: cfg}
}

func (m *BasicAuth) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		var lastError error
		l := len(basic)
		for _, auth := range c.Req().Header.Values(wool.HeaderAuthorization) {
			if !(len(auth) > l+1 && strings.EqualFold(auth[:l], basic)) {
				continue
			}

			// Invalid base64 shouldn't be treated as error
			// instead should be treated as invalid client input
			b, errDecode := base64.StdEncoding.DecodeString(auth[l+1:])
			if errDecode != nil {
				lastError = wool.NewErrBadRequest(errDecode)
				continue
			}
			idx := bytes.IndexByte(b, ':')
			if idx >= 0 {
				valid, errValidate := m.cfg.Validator(c, string(b[:idx]), string(b[idx+1:]))
				if errValidate != nil {
					lastError = errValidate
				} else if valid {
					return next(c)
				}
			}
		}

		if lastError != nil {
			return lastError
		}

		c.Res().Header().Set(wool.HeaderWWWAuthenticate, basic+" realm="+m.cfg.Realm)
		return wool.NewErrUnauthorized(nil)
	}
}
