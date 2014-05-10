TurtleProxy simulates slow connections
======================================

Currently TurtleProxy supports mobile connections as listed in --help

You can also assign your own speed and latency.

Run
---

You can access the binaries prebuilt here:

[releases/tag/0.1](https://github.com/olafura/turtleproxy/releases/tag/0.1)

If you want to compile it yourself then it's easy. It's written in Go so you need to execute those commands:

`go get`

`go install`

Instead of installing it you can also do:

`go run turtleproxy.go`

Read [How to Write Go Code](http://golang.org/doc/code.html)
to get the layout of your workspace right

Future
------
* Add support for latency ranges
* Add other connections to simulate
* Possible introduce package loss

