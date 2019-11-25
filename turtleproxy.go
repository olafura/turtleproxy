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
	speedStart uint64
	speedEnd   uint64
	startTime  time.Time
	bytes      int64
}

func (c *DelayReadCloser) Read(b []byte) (n int, err error) {
	n, err = c.R.Read(b)
	c.bytes += int64(n)
	return
}
func (c DelayReadCloser) Close() error {
	endTime := time.Now()
	timepassed := endTime.Sub(c.startTime)
	var speed int64
	if c.speedEnd == 0 {
		speed = int64(c.speedStart)
	} else {
		speed = randRange(c.speedStart, c.speedEnd)
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

func randRange(min, max uint64) int64 {
	rand.Seed(time.Now().Unix())
	return rand.Int63n(int64(max-min)) + int64(min)
}

func main() {
	verboseArg := flag.Bool("v", false, "Print out all messages")
	speedHumanArg := flag.String("s", "808Kb", "Speed of the connection")
	latencyArg := flag.Int64("l", 200, "Latency of connection in ms")
	conntext := `Type of connection
	 Available:
	  "gsm"
	  "gprs"
	  "edge"
	  "umts"
	  "hspa"
	  "lte"`
	connectionArg := flag.String("c", "", conntext)
	addrArg := flag.String("addr", ":8080", "proxy listen address")
	flag.Parse()
	if *verboseArg == false {
		log.SetOutput(ioutil.Discard)
	}
	if *connectionArg != "" {
		connType, err1 := Connections[strings.ToLower(*connectionArg)]
		if err1 == false {
			log.Fatal("Type of connection not found: ", *connectionArg)
		}
		if connType.SpeedEnd != "" {
			speedTemp := ""
			speedTemp += connType.SpeedStart
			speedTemp += "-"
			speedTemp += connType.SpeedEnd
			speedHumanArg = &speedTemp
		} else {
			speedHumanArg = &connType.SpeedStart
		}
		latencyArg = &connType.Latency
	}
	log.Println("speed", *speedHumanArg)
	log.Println("latency", *latencyArg)
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verboseArg
	speedHumanValues := strings.Split(*speedHumanArg, "-")
	var speedStart uint64 = 0
	var speedEnd uint64 = 0
	var err2, err3, err4 error
	if len(speedHumanValues) > 1 {
		speedStart, err2 = humanize.ParseBytes(speedHumanValues[0])
		if err2 != nil {
			log.Fatal(err2)
		}
		speedEnd, err3 = humanize.ParseBytes(speedHumanValues[1])
		if err3 != nil {
			log.Fatal(err3)
		}
	} else {
		speedStart, err4 = humanize.ParseBytes(*speedHumanArg)
		if err4 != nil {
			log.Fatal(err4)
		}
	}
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		time.Sleep(time.Duration(*latencyArg) * time.Millisecond)
		startTime := time.Now()
		resp.Body = &DelayReadCloser{resp.Body, speedStart, speedEnd, startTime, 0}
		return resp
	})
	log.Fatal(http.ListenAndServe(*addrArg, proxy))
}
