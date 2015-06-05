/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubectl

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
)

// ProxyServer is a http.Handler which proxies Kubernetes APIs to remote API server.
type ProxyServer struct {
	mux *http.ServeMux
	httputil.ReverseProxy
}

// NewProxyServer creates and installs a new ProxyServer.
// It automatically registers the created ProxyServer to http.DefaultServeMux.
func NewProxyServer(filebase string, apiProxyPrefix string, staticPrefix string, cfg *client.Config) (*ProxyServer, error) {
	host := cfg.Host
	if !strings.HasSuffix(host, "/") {
		host = host + "/"
	}
	target, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	proxy := newProxyServer(target)
	if proxy.Transport, err = client.TransportFor(cfg); err != nil {
		return nil, err
	}
	if strings.HasPrefix(apiProxyPrefix, "/api") {
		proxy.mux.Handle(apiProxyPrefix, proxy)
	} else {
		proxy.mux.Handle(apiProxyPrefix, http.StripPrefix(apiProxyPrefix, proxy))
	}
	proxy.mux.Handle(staticPrefix, newFileHandler(staticPrefix, filebase))
	return proxy, nil
}

// Serve starts the server (http.DefaultServeMux) on given port, loops forever.
func (s *ProxyServer) Serve(port int) error {
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.mux,
	}
	return server.ListenAndServe()
}

func newProxyServer(target *url.URL) *ProxyServer {
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
	}
	return &ProxyServer{
		ReverseProxy: httputil.ReverseProxy{Director: director},
		mux:          http.NewServeMux(),
	}
}

func newFileHandler(prefix, base string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(base)))
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
