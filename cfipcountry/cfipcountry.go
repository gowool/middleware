package cfipcountry

import (
	"github.com/dlclark/regexp2"
	"github.com/gowool/wool"
	"net/http"
)

const HeaderCfIPCountry = "CF-IPCountry"

type Config map[string]*struct {
	Pattern     string `mapstructure:"pattern"`
	RedirectURL string `mapstructure:"redirect_url"`
	Permanently bool   `mapstructure:"permanently"`
	compiled    *regexp2.Regexp
}

func (cfg *Config) init() {
	for _, v := range *cfg {
		if v.Pattern == "" {
			panic("pattern is empty")
		}
		re, err := regexp2.Compile(v.Pattern, regexp2.IgnoreCase&regexp2.RE2)
		if err != nil {
			panic(err)
		}
		v.compiled = re
		if v.RedirectURL == "" {
			v.RedirectURL = "/"
		}
	}
}

type CfIPCountry struct {
	cfg *Config
}

func Middleware(cfg *Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg *Config) *CfIPCountry {
	if cfg != nil {
		cfg.init()
	}
	return &CfIPCountry{cfg: cfg}
}

func (m *CfIPCountry) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		country := c.Req().Header.Get(HeaderCfIPCountry)

		if m.cfg == nil || len(*m.cfg) == 0 {
			return next(c)
		}

		url := c.Req().URL.String()

		if item, ok := (*m.cfg)[country]; ok {
			if ok, _ = item.compiled.MatchString(url); !ok {
				if item.Permanently {
					return c.Redirect(http.StatusMovedPermanently, item.RedirectURL)
				}
				return c.Redirect(http.StatusFound, item.RedirectURL)
			}
		}

		return next(c)
	}
}
