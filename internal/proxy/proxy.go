package proxy

import (
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// NewReverseProxy forwards requests under a gin *path param to target.
// ponytail: hardcoded targets at call sites, move to env vars if hosts move off-box
func NewReverseProxy(target string) gin.HandlerFunc {
	u, err := url.Parse(target)
	if err != nil {
		panic(err) // programmer error, target is a constant
	}
	rp := httputil.NewSingleHostReverseProxy(u)

	return func(c *gin.Context) {
		c.Request.URL.Path = c.Param("path")
		rp.ServeHTTP(c.Writer, c.Request)
	}
}
