package server

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/cloudwego/hertz/pkg/app"
)

// statusRecorder captures the output of a net/http handler.
type statusRecorder struct {
	code   int
	body   bytes.Buffer
	header http.Header
}

func (r *statusRecorder) Header() http.Header { return r.header }
func (r *statusRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}
func (r *statusRecorder) WriteHeader(code int) { r.code = code }

// serveNetHTTP executes a standard net/http handler against the Hertz
// request context. Used to mount swaggo/http-swagger and Prometheus.
func serveNetHTTP(c *app.RequestContext, handler http.Handler) {
	req := buildHTTPRequest(c)
	rec := &statusRecorder{code: 200, header: http.Header{}}
	handler.ServeHTTP(rec, req)
	for k, vv := range rec.header {
		for _, v := range vv {
			c.Response.Header.Add(k, v)
		}
	}
	c.Response.SetStatusCode(rec.code)
	c.Response.SetBody(rec.body.Bytes())
}

// buildHTTPRequest converts the hertz request into a net/http request.
func buildHTTPRequest(c *app.RequestContext) *http.Request {
	req := &http.Request{
		Method:     string(c.Method()),
		URL:        &url.URL{Path: string(c.Path()), RawQuery: string(c.Request.URI().QueryString())},
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(c.Request.Body())),
		Host:       string(c.Host()),
		RequestURI: string(c.Request.URI().RequestURI()),
	}
	c.Request.Header.VisitAll(func(k, v []byte) {
		req.Header.Add(string(k), string(v))
	})
	return req
}
