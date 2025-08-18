package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elazarl/goproxy"
)

type DelayReadWriteCloser struct {
	R          io.ReadWriteCloser
	speedStart uint64
	speedEnd   uint64
	startTime  time.Time
	readBytes  int64
	writeBytes int64
}

type DelayReadCloser struct {
	R          io.ReadCloser
	speedStart uint64
	speedEnd   uint64
	startTime  time.Time
	bytes      int64
}

func (c *DelayReadWriteCloser) Read(b []byte) (n int, err error) {
	logger := slog.Default().With("epoch", c.startTime.Unix())
	logger.Info("websocket read")
	delay := delay(*logger, int64(len(b)), c.speedStart, c.speedEnd)
	time.Sleep(delay)

	n, err = c.R.Read(b)
	c.readBytes += int64(n)
	return
}

func (c *DelayReadWriteCloser) Write(b []byte) (n int, err error) {
	logger := slog.Default().With("epoch", c.startTime.Unix())
	logger.Info("websocket write")
	delay := delay(*logger, int64(len(b)), c.speedStart, c.speedEnd)
	time.Sleep(delay)

	n, err = c.R.Write(b)
	c.writeBytes += int64(n)
	return
}

func (c DelayReadWriteCloser) Close() error {
	logger := slog.Default().With("epoch", c.startTime.Unix())
	logger.Info("websocket close")
	endTime := time.Now()
	timepassed := endTime.Sub(c.startTime)

	timepassedSec := timepassed.Seconds()
	if timepassedSec > 0 {
		readSpeed := float64(c.readBytes) / float64(timepassedSec)
		logger.Info(fmt.Sprintf("effective read speed: %s/s", humanize.Bytes(uint64(readSpeed))))
		writeSpeed := float64(c.writeBytes) / float64(timepassedSec)
		logger.Info(fmt.Sprintf("effective write speed: %s/s", humanize.Bytes(uint64(writeSpeed))))
	}
	logger.Info(fmt.Sprintf("total bytes: %s", humanize.Bytes(uint64(c.readBytes+c.writeBytes))))
	logger.Info(fmt.Sprintf("timepassed: %v", timepassed))

	return c.R.Close()
}

func (c *DelayReadCloser) Read(b []byte) (n int, err error) {
	logger := slog.Default().With("epoch", c.startTime.Unix())
	delay := delay(*logger, int64(len(b)), c.speedStart, c.speedEnd)
	time.Sleep(delay)

	n, err = c.R.Read(b)
	c.bytes += int64(n)
	return
}

func (c DelayReadCloser) Close() error {
	logger := slog.Default().With("epoch", c.startTime.Unix())
	endTime := time.Now()
	timepassed := endTime.Sub(c.startTime)

	timepassedSec := timepassed.Seconds()
	if timepassedSec > 0 {
		speed := float64(c.bytes) / float64(timepassedSec)
		logger.Info(fmt.Sprintf("effective speed: %s/s", humanize.Bytes(uint64(speed))))
	}
	logger.Info(fmt.Sprintf("total bytes: %s", humanize.Bytes(uint64(c.bytes))))
	logger.Info(fmt.Sprintf("timepassed: %v", timepassed))

	return c.R.Close()
}

func delay(logger slog.Logger, bytes int64, speedStart uint64, speedEnd uint64) time.Duration {
	var speed uint64
	if speedEnd == 0 {
		speed = speedStart
	} else {
		speed = randRange(speedStart, speedEnd)
	}

	delay := time.Duration(float64(bytes)*8/float64(speed)*1000) * time.Millisecond

	logger.Info(fmt.Sprintf("bytes: %s", humanize.Bytes(uint64(bytes))))
	logger.Info(fmt.Sprintf("speed: %s/s", humanize.Bytes(speed)))
	logger.Info(fmt.Sprintf("delay: %v", delay))

	return delay
}

type BetterLogger struct{}

func (b BetterLogger) Printf(format string, v ...any) {
	logger := slog.Default()
	logger.Info(fmt.Sprintf(format, v...))
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

func randRange(min, max uint64) uint64 {
	diff := max - min
	return uint64(rand.Int63n(int64(diff))) + min
}

func main() {
	verboseArg := flag.Bool("v", false, "Print out all messages")
	useCert := flag.Bool("usecert", true, "Use cert for for https")
	caRoot := flag.String("caroot", "", "Path to the CA root directory, default is ~/.local/share/mkcert or CAROOT")
	jsonLog := flag.Bool("json", false, "Use json for logging")
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

	if !*verboseArg {
		log.SetOutput(io.Discard)
	}

	if *jsonLog {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger)
	}

	if *connectionArg != "" {
		connType, err1 := Connections[strings.ToLower(*connectionArg)]
		if !err1 {
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

	log.Printf("speed: %s/s\n", *speedHumanArg)
	log.Println("latency: ", *latencyArg)

	proxy := goproxy.NewProxyHttpServer()

	proxy.Logger = &BetterLogger{}

	if *useCert {
		cert, err := GetCert(*caRoot)

		if err == nil {
			log.Println("Using cert for https")
			proxy.CertStore = NewCertStorage()
			customCaMitm := &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(cert)}
			var customAlwaysMitm goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				return customCaMitm, host
			}
			proxy.OnRequest().HandleConnect(customAlwaysMitm)
		} else {
			log.Println("Not using cert for https, error:", err)
		}
	}

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
		startTime := time.Now()
		logger := slog.Default().With("epoch", startTime.Unix())
		// We don't want to mess with Websocket upgrade and the like
		if resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if resp.Request != nil && resp.Request.URL != nil {
				logger.Info(fmt.Sprintf("Response received for: %s", resp.Request.URL))

			}
			time.Sleep(time.Duration(*latencyArg) * time.Millisecond)
			resp.Body = &DelayReadCloser{resp.Body, speedStart, speedEnd, startTime, 0}
		} else {
			isWebsocket := isWebSocketHandshake(resp.Header)
			if isWebsocket {
				if resp.Request != nil && resp.Request.URL != nil {
					logger.Info(fmt.Sprintf("Response received for: %s", resp.Request.URL))

				}
				time.Sleep(time.Duration(*latencyArg) * time.Millisecond)
				resp.Body = &DelayReadWriteCloser{resp.Body.(io.ReadWriteCloser), speedStart, speedEnd, startTime, 0, 0}
			}
		}
		return resp
	})

	log.Fatal(http.ListenAndServe(*addrArg, proxy))
}
