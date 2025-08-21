package turtleproxy

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elazarl/goproxy"
	"github.com/olafura/turtleproxy/internal/cache"
	"github.com/olafura/turtleproxy/internal/cert"
	"github.com/olafura/turtleproxy/internal/websocket"
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
	totalBytes := c.readBytes + c.writeBytes
	var totalBytesUint64 uint64
	if totalBytes < 0 {
		totalBytesUint64 = 0
	} else {
		totalBytesUint64 = uint64(totalBytes)
	}
	logger.Info(fmt.Sprintf("total bytes: %s", humanize.Bytes(totalBytesUint64)))
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
	if c.bytes < 0 {
		c.bytes = 0
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

	delay := time.Duration((float64(bytes)/float64(speed))*1000) * time.Millisecond

	var bytesUint64 uint64
	if bytes < 0 {
		bytesUint64 = 0
	} else {
		bytesUint64 = uint64(bytes)
	}
	logger.Info(fmt.Sprintf("bytes: %s", humanize.Bytes(bytesUint64)))
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

const MaxUint64 = ^uint64(0)

func randRange(min, max uint64) uint64 {
	diff := max - min
	if diff == 0 {
		return min
	}

	if diff > MaxUint64 {
		return min
	}

	diffInt := int64(diff) // #nosec G115 - diff is bounded above

	randNum, err := rand.Int(rand.Reader, big.NewInt(diffInt))
	if err != nil {
		// Fallback to min if crypto rand fails
		return min
	}

	randInt64 := randNum.Int64()
	if randInt64 < 0 || randInt64 > diffInt {
		randInt64 = 0
	}

	randUint64 := uint64(randInt64) + min // #nosec G115 - randInt64 is validated above

	if randUint64 < min || randUint64 > max {
		return min
	}

	return randUint64
}

func sanitizeURL(u *url.URL) string {
	if u == nil {
		return "[nil URL]"
	}

	sanitized := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}

	return sanitized.String()
}

func Proxy(addr string, useCert bool, caRoot string, verbose bool, latency int64, speedStart uint64, speedEnd uint64) {
	logger := slog.Default()

	proxy := goproxy.NewProxyHttpServer()

	proxy.Logger = &BetterLogger{}

	if useCert {
		cert, err := cert.GetCert(caRoot)

		if err == nil {
			logger.Info("Using cert for https")
			proxy.CertStore = cache.NewCertStorage()
			customCaMitm := &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(cert)}
			var customAlwaysMitm goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				return customCaMitm, host
			}
			proxy.OnRequest().HandleConnect(customAlwaysMitm)
		} else {
			logger.Error("Not using cert for https", "error", err)
		}
	}

	proxy.Verbose = verbose

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		startTime := time.Now()
		logger := slog.Default().With("epoch", startTime.Unix())
		// We don't want to mess with Websocket upgrade and the like
		if resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if resp.Request != nil && resp.Request.URL != nil {
				logger.Info(fmt.Sprintf("Response received for: %s", sanitizeURL(resp.Request.URL)))
			}
			time.Sleep(time.Duration(latency) * time.Millisecond)
			resp.Body = &DelayReadCloser{resp.Body, speedStart, speedEnd, startTime, 0}
		} else {
			isWebsocket := websocket.IsWebSocketHandshake(resp.Header)
			if isWebsocket {
				if resp.Request != nil && resp.Request.URL != nil {
					logger.Info(fmt.Sprintf("Response received for: %s", sanitizeURL(resp.Request.URL)))
				}
				time.Sleep(time.Duration(latency) * time.Millisecond)

				if rwc, ok := resp.Body.(io.ReadWriteCloser); ok {
					resp.Body = &DelayReadWriteCloser{rwc, speedStart, speedEnd, startTime, 0, 0}
				} else {
					logger.Info("WebSocket upgrade response body does not implement ReadWriteCloser, using standard DelayReadCloser")
					resp.Body = &DelayReadCloser{resp.Body, speedStart, speedEnd, startTime, 0}
				}
			}
		}
		return resp
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
