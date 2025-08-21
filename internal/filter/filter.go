package filter

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
)

// adapted from https://github.com/elazarl/goproxy/blob/master/dispatcher.go

func RespHostMatches(regexps ...*regexp.Regexp) goproxy.RespCondition {
	return goproxy.RespConditionFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
		for _, re := range regexps {
			if re.MatchString(resp.Request.Host + ":" + resp.Request.URL.Port()) {
				return true
			}
		}
		return false
	})
}

func DstHostIs(host string) goproxy.RespCondition {
	// Make sure to perform a case-insensitive host check
	host = strings.ToLower(host)
	var port string

	// Check if the user specified a custom port that we need to match
	if strings.Contains(host, ":") {
		hostOnly, portOnly, err := net.SplitHostPort(host)
		if err == nil {
			host = hostOnly
			port = portOnly
		}
	}

	return goproxy.RespConditionFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
		// Check port matching only if it was specified
		if port != "" && port != resp.Request.URL.Port() {
			return false
		}

		return strings.ToLower(resp.Request.URL.Hostname()) == host
	})
}

func URLMatches(re *regexp.Regexp) goproxy.RespCondition {
	return goproxy.RespConditionFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
		return re.MatchString(resp.Request.URL.Path) ||
			re.MatchString(resp.Request.URL.Host+resp.Request.URL.Path)
	})
}

func GetConds(filterHost, regexHost, regexURL string) []goproxy.RespCondition {
	logger := slog.Default()

	conds := []goproxy.RespCondition{}

	if filterHost != "" {
		logger.Info(fmt.Sprintf("Add host filtering %s", filterHost))
		conds = append(conds, DstHostIs(filterHost))
	}

	if regexHost != "" {
		logger.Info(fmt.Sprintf("Add regex host filtering %s", regexHost))
		conds = append(conds, RespHostMatches(regexp.MustCompile(regexHost)))
	}

	if regexURL != "" {
		logger.Info(fmt.Sprintf("Add regex URL filtering %s", regexURL))
		conds = append(conds, URLMatches(regexp.MustCompile(regexURL)))
	}
	return conds
}
