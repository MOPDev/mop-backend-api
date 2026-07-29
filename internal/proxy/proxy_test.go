package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReverseProxyForwardsPathAndQuery(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/" || r.URL.Query().Get("q") != "Hvidovre" {
			t.Fatalf("unexpected upstream request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/geocode/*path", NewReverseProxy(upstream.URL))

	testSrv := httptest.NewServer(r)
	defer testSrv.Close()

	resp, err := http.Get(testSrv.URL + "/geocode/api/?q=Hvidovre")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("expected 'ok', got %q", body)
	}
}
