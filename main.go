package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/nanreh/portpusher/internal/env"
	"github.com/nanreh/portpusher/internal/gluetun"
	"github.com/nanreh/portpusher/internal/logging"
)

type PortPusher interface {
	Push(int) error
}

func main() {
	logLevel, err := env.GetLogLevel()
	if err != nil {
		fmt.Println(err)
		return
	}
	logger := logging.NewLogger(logLevel)

	delayError, err := env.GetDelayError()
	if err != nil {
		fmt.Println(err)
		return
	}

	delaySuccess, err := env.GetDelaySuccess()
	if err != nil {
		fmt.Println(err)
		return
	}

	jar, err := cookiejar.New(nil)
	if nil != err {
		fmt.Printf("%s\n", err)
		return
	}
	httpClient := &http.Client{
		Jar: jar,
	}

	gtc, err := env.GetGluetunClient(httpClient, logger)
	if err != nil {
		logger.Error("Error building Gluetun client: %v", err)
		return
	}

	pushers := make([]PortPusher, 0, 3)

	tc, err := env.GetTransmissionClient(httpClient, logger)
	if err != nil {
		logger.Error("Error building Transmission client: %v", err)
		return
	} else if tc != nil {
		pushers = append(pushers, tc)
	}

	qbt, err := env.GetQbittorrentClient(httpClient, logger)
	if err != nil {
		logger.Error("Error building QBittorrent client: %v", err)
		return
	} else if qbt != nil {
		pushers = append(pushers, qbt)
	}

	dc, err := env.GetDelugeClient(httpClient, logger)
	if err != nil {
		logger.Error("Error building Deluge client: %v", err)
		return
	} else if dc != nil {
		pushers = append(pushers, dc)
	}

	if len(pushers) == 0 {
		logger.Error("No bittorrent clients are configured, nothing to do")
		return
	}

	loop(logger, gtc, pushers, delaySuccess, delayError)
}

func loop(logger logging.Logger, gtc *gluetun.Client, pushers []PortPusher, delaySuccess time.Duration, delayError time.Duration) {
	for {
		logger.Info("Running...")
		// fetch forwarded port
		port, err := gtc.PullPort()
		if err != nil {
			logger.Info("Done. Next push attempt in %v.", delayError)
			time.Sleep(delayError)
		} else {
			isError := false
			for _, p := range pushers {
				// push forwarded port
				err = p.Push(port)
				if err != nil {
					isError = true
				}
			}
			if isError {
				logger.Info("Done. Next push attemptin %v.", delaySuccess)
				time.Sleep(delaySuccess)
			} else {
				logger.Info("Done. Next push in %v.", delayError)
				time.Sleep(delayError)
			}
		}
	}
}
