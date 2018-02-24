Judas
=====
Judas is a phishing proxy.
It can clone a website passed to it using command line flags.

Usage
-----
The only required command line flag is ```--target```.

Example:
```
./proxy --target https://target-url.com
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