package proxy

import (
	"context"
	"net/http"
	"regexp"

	"github.com/elazarl/goproxy"
)

type KeyGetter interface {
	UseBestKey(ctx context.Context) (*string, error)
}

func CreateProxyHandler(ctx context.Context, keyGetter KeyGetter) http.Handler {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(
		goproxy.ReqHostMatches(regexp.MustCompile(".*")),
	).DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			key, err := keyGetter.UseBestKey(r.Context())
			if err == nil && key != nil {
				r.Header.Set("Authorization", "Bearer "+*key)
			}
			return r, nil
		})

	return proxy
}
