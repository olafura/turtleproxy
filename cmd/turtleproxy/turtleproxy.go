package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/olafura/turtleproxy/pkg/turtleproxy"
)

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

func main() {
	verboseArg := flag.Bool("v", false, "Print out all messages")
	useCertArg := flag.Bool("usecert", true, "Use cert for for https")
	caRootArg := flag.String("caroot", "", "Path to the CA root directory, default is ~/.local/share/mkcert or CAROOT")
	jsonLogArg := flag.Bool("json", false, "Use json for logging")
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

	if *jsonLogArg {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	logger := slog.Default()

	if *connectionArg != "" {
		connTypeKey := strings.ToLower(strings.TrimSpace(*connectionArg))
		if connTypeKey == "" {
			logger.Error("Connection type cannot be empty")
		}
		connType, exists := Connections[connTypeKey]
		if !exists {
			logger.Error(fmt.Sprintf("Type of connection not found: %s", *connectionArg))
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

	logger.Info(fmt.Sprintf("speed: %s/s", *speedHumanArg))
	logger.Info(fmt.Sprintf("latency: %v", *latencyArg))

	speedHumanValues := strings.Split(*speedHumanArg, "-")

	var speedStart uint64
	var speedEnd uint64 = 0
	var err2, err3, err4 error

	if len(speedHumanValues) > 1 {
		speedStart, err2 = humanize.ParseBytes(speedHumanValues[0])
		if err2 != nil {
			logger.Error(fmt.Sprintf("%v", err2))
		}
		speedEnd, err3 = humanize.ParseBytes(speedHumanValues[1])
		if err3 != nil {
			logger.Error(fmt.Sprintf("%v", err3))
		}
	} else {
		speedStart, err4 = humanize.ParseBytes(*speedHumanArg)
		if err4 != nil {
			logger.Error(fmt.Sprintf("%v", err4))
		}
	}
	turtleproxy.Proxy(*addrArg, *useCertArg, *caRootArg, *verboseArg, *latencyArg, speedStart, speedEnd)
}
