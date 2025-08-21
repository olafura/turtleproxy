# TurtleProxy simulates slow connections

Currently TurtleProxy supports mobile connections as listed in --help

You can also assign your own speed and latency.

The heavy lifting is done by the excellent [goproxy](https://github.com/elazarl/goproxy)

## Run

You can access the binaries prebuilt here:

[releases/tag/v0.2](https://github.com/olafura/turtleproxy/releases/tag/v0.2)

If you want to compile it yourself then it's easy. It's written in Go so you need to execute those commands:

`go get`

`go install`

Instead of installing it you can also do:

`go run turtleproxy.go`


## Https support

#### MkCert
Do you want a managed, easy to use solution that automatically generates
a root CA certificate for local usage, and automatically adds it to the trusted system
certificates? Consider [MkCert](https://github.com/FiloSottile/mkcert).

Future
------
* Add support for latency ranges
* Add other connections to simulate
* Possible introduce package loss

