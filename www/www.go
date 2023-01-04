package www

import (
	"github.com/gowool/wool"
	"net/http"
	"strings"
)

type WWW struct {
}

func Middleware() wool.Middleware {
	return New().Middleware
}

func New() *WWW {
	return &WWW{}
}

func (*WWW) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		if data := strings.Split(c.Req().Host, "."); len(data) == 2 {
			c.Req().URL.Host = "www." + c.Req().Host
			return c.Redirect(http.StatusMovedPermanently, c.Req().URL.String())
		}
		return next(c)
	}
}
