package proxy

import (
	"net/http"
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
	rp.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin") // ponytail: drop upstream's own CORS header, let our middleware set one
		return nil
	}

	return func(c *gin.Context) {
		c.Request.URL.Path = c.Param("path")
		c.Request.Header.Set("X-Forwarded-Path", "/api/v1/tiles")
		rp.ServeHTTP(c.Writer, c.Request)
	}
}
