package main

import (
	"flag"
	"github.com/dustin/go-humanize"
	"github.com/elazarl/goproxy"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type DelayReadCloser struct {
	R          io.ReadCloser
	speedstart uint64
	speedend   uint64
	starttime  time.Time
	bytes      int64
}

func (c *DelayReadCloser) Read(b []byte) (n int, err error) {
	n, err = c.R.Read(b)
	c.bytes += int64(n)
	return
}
func (c DelayReadCloser) Close() error {
	endtime := time.Now()
	timepassed := endtime.Sub(c.starttime)
	var speed int64
	if c.speedend == 0 {
		speed = int64(c.speedstart)
	} else {
		speed = randrange(c.speedstart, c.speedend)
	}
	delay := time.Duration(float64(c.bytes)*8/float64(speed)*1000) * time.Millisecond
	log.Println("bytes: ", c.bytes)
	log.Println("speed: ", speed)
	log.Println("delay: ", delay)
	log.Println("timepassed: ", timepassed)
	time.Sleep(delay - timepassed)
	return c.R.Close()
}

type Conn struct {
	SpeedStart string
	SpeedEnd   string
	Latency    int64
}

type ConnMap map[string]Conn

var Connections = ConnMap{
	"gsm":  Conn{"9.6Kb", "", 650},
	"gprs": Conn{"35Kb", "171Kb", 650},
	"edge": Conn{"120Kb", "384Kb", 300},
	"umts": Conn{"384Kb", "2Mb", 200},
	"hspa": Conn{"600Kb", "10Mb", 100},
	"lte":  Conn{"3Mb", "10Mb", 50},
}

func randrange(min, max uint64) int64 {
	rand.Seed(time.Now().Unix())
	return rand.Int63n(int64(max-min)) + int64(min)
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
		if conn.SpeedEnd != "" {
			speedtemp := ""
			speedtemp += conn.SpeedStart
			speedtemp += "-"
			speedtemp += conn.SpeedEnd
			speedhuman = &speedtemp
		} else {
			speedhuman = &conn.SpeedStart
		}
		latency = &conn.Latency
	}
	log.Println("speed", *speedhuman)
	log.Println("latency", *latency)
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	speedhumanvalues := strings.Split(*speedhuman, "-")
	var speed1 uint64 = 0
	var speed2 uint64 = 0
	var err2, err3, err4 error
	if len(speedhumanvalues) > 1 {
		speed1, err2 = humanize.ParseBytes(speedhumanvalues[0])
		if err2 != nil {
			log.Fatal(err2)
		}
		speed2, err3 = humanize.ParseBytes(speedhumanvalues[1])
		if err3 != nil {
			log.Fatal(err3)
		}
	} else {
		speed1, err4 = humanize.ParseBytes(*speedhuman)
		if err4 != nil {
			log.Fatal(err4)
		}
	}
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		time.Sleep(time.Duration(*latency) * time.Millisecond)
		starttime := time.Now()
		resp.Body = &DelayReadCloser{resp.Body, speed1, speed2, starttime, 0}
		return resp
	})
	log.Fatal(http.ListenAndServe(*addr, proxy))
}
