package main

import (
	"flag"
	"github.com/dustin/go-humanize"
	"github.com/elazarl/goproxy"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type DelayReadCloser struct {
	R         io.ReadCloser
	speed     uint64
	starttime time.Time
	bytes        int64
}

func (c *DelayReadCloser) Read(b []byte) (n int, err error) {
	n, err = c.R.Read(b)
	c.bytes += int64(n)
	return
}
func (c DelayReadCloser) Close() error {
	endtime := time.Now()
	timepassed := endtime.Sub(c.starttime)
	delay := time.Duration(float64(c.bytes)*8/float64(c.speed)*1000) * time.Millisecond
	log.Println("bytes: ", c.bytes)
	log.Println("speed: ", c.speed)
	log.Println("delay: ", delay)
	log.Println("timepassed: ", timepassed)
	time.Sleep(delay - timepassed)
	return c.R.Close()
}

type Conn struct {
	Speed   string
	Latency int64
}

type ConnMap map[string]Conn

var Connections = ConnMap{
	"gsm":  Conn{"9.6Kb", 650},
	"gprs": Conn{"103Kb", 650},
	"edge": Conn{"132Kb", 300},
	"umts": Conn{"808Kb", 200},
	"hspa": Conn{"2Mb", 100},
	"lte":  Conn{"3.5Mb", 50},
}

func main() {
	verbose := flag.Bool("v", false, "Print out all messages")
	speedhuman := flag.String("s", "808Kb", "Speed of the connection")
	latency := flag.Int64("l", 200, "Latency of connection in ms")
	conntext := `Type of connection
	 Available:
	  "gsm"
	  "gprs"
	  "edge"
	  "umts"
	  "hspa"
	  "lte"`
	connection := flag.String("c", "", conntext)
	addr := flag.String("addr", ":8080", "proxy listen address")
	flag.Parse()
	if *verbose == false {
		log.SetOutput(ioutil.Discard)
	}
	if *connection != "" {
		conn, err1 := Connections[strings.ToLower(*connection)]
		if err1 == false {
			log.Fatal("Type of connection not found: ", *connection)
		}
		speedhuman = &conn.Speed
		latency = &conn.Latency
	}
	log.Println("speed", *speedhuman)
	log.Println("latency", *latency)
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	speed, err2 := humanize.ParseBytes(*speedhuman)
	if err2 != nil {
		log.Fatal(err2)
	} else {
		proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			time.Sleep(time.Duration(*latency) * time.Millisecond)
			starttime := time.Now()
			resp.Body = &DelayReadCloser{resp.Body, speed, starttime, 0}
			return resp
		})
		log.Fatal(http.ListenAndServe(*addr, proxy))
	}
}
