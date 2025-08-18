package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elazarl/goproxy"
)

type DelayReadCloser struct {
	R          io.ReadCloser
	speedStart uint64
	speedEnd   uint64
	startTime  time.Time
	bytes      int64
}

func (c *DelayReadCloser) Read(b []byte) (n int, err error) {
	delay := delay(int64(len(b)), c.speedStart, c.speedEnd)
	time.Sleep(delay)

	n, err = c.R.Read(b)
	c.bytes += int64(n)
	return
}

func (c DelayReadCloser) Close() error {
	endTime := time.Now()
	timepassed := endTime.Sub(c.startTime)

	timepassedSec := timepassed.Seconds()
	if timepassedSec > 0 {
		speed := float64(c.bytes) / float64(timepassedSec)
		log.Printf("effective speed: %s/s \n", humanize.Bytes(uint64(speed)))
	}
	log.Println("total bytes: ", humanize.Bytes(uint64(c.bytes)))
	log.Println("timepassed: ", timepassed)

	return c.R.Close()
}

func delay(bytes int64, speedStart uint64, speedEnd uint64) time.Duration {
	var speed uint64
	if speedEnd == 0 {
		speed = speedStart
	} else {
		speed = randRange(speedStart, speedEnd)
	}

	delay := time.Duration(float64(bytes)*8/float64(speed)*1000) * time.Millisecond

	log.Println("bytes: ", humanize.Bytes(uint64(bytes)))
	log.Printf("speed: %s/s \n", humanize.Bytes(speed))
	log.Println("delay: ", delay)

	return delay
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

func getCA(caRoot string) (*tls.Certificate, error) {
	var (
		err        error
		caCert     []byte
		caCertKey  []byte
		parsedCert tls.Certificate
	)

	if caRoot == "" {
		caRoot = os.Getenv("CAROOT")
	}

	if caRoot == "" {
		caRoot = "~/.local/share/mkcert"
	}

	if strings.HasPrefix(caRoot, "~/") {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}

		caRoot = filepath.Join(usr.HomeDir, caRoot[2:])
	}

	_, err = os.Stat(caRoot)

	if err != nil {
		return nil, err
	}

	rootCAPath := filepath.Join(caRoot, "rootCA.pem")
	rootCAKeyPath := filepath.Join(caRoot, "rootCA-key.pem")

	_, err = os.Stat(rootCAPath)

	if err != nil {
		return nil, err
	}

	_, err = os.Stat(rootCAKeyPath)

	if err != nil {
		return nil, err
	}

	caCert, err = os.ReadFile(rootCAPath)

	if err != nil {
		return nil, err
	}

	caCertKey, err = os.ReadFile(rootCAKeyPath)

	if err != nil {
		return nil, err
	}

	parsedCert, err = tls.X509KeyPair(caCert, caCertKey)

	if err != nil {
		return nil, err
	}

	if parsedCert.Leaf, err = x509.ParseCertificate(parsedCert.Certificate[0]); err != nil {
		return nil, err
	}

	return &parsedCert, nil
}

func main() {
	verboseArg := flag.Bool("v", false, "Print out all messages")
	useCert := flag.Bool("usecert", true, "Use cert for for https")
	caRoot := flag.String("caroot", "", "Path to the CA root directory, default is ~/.local/share/mkcert or CAROOT")
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

	if *useCert {
		cert, err := getCA(*caRoot)

		if err == nil {
			log.Println("Using cert for https")
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
		log.Println("Response received for:", resp.Request.URL)
		time.Sleep(time.Duration(*latencyArg) * time.Millisecond)
		startTime := time.Now()
		resp.Body = &DelayReadCloser{resp.Body, speedStart, speedEnd, startTime, 0}
		return resp
	})

	log.Fatal(http.ListenAndServe(*addrArg, proxy))
}
