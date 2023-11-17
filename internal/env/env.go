package env

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nanreh/portpusher/internal/deluge"
	"github.com/nanreh/portpusher/internal/gluetun"
	"github.com/nanreh/portpusher/internal/logging"
	"github.com/nanreh/portpusher/internal/qbittorrent"
	"github.com/nanreh/portpusher/internal/transmission"
)

const (
	envLogLevel            = "PUSHER_LOG_LEVEL"
	envDelayError          = "PUSHER_DELAY_ERROR"
	envDelaySuccess        = "PUSHER_DELAY_SUCCESS"
	envGluetunHost         = "GLUETUN_HOST"
	envGluetunPort         = "GLUETUN_PORT"
	envTransmissionEnabled = "TRANSMISSION_ENABLED"
	envTransmissionHost    = "TRANSMISSION_HOST"
	envTransmissionPort    = "TRANSMISSION_PORT"
	envTransmissionUser    = "TRANSMISSION_USER"
	envTransmissionPass    = "TRANSMISSION_PASS"
	envQbittorrentEnabled  = "QBITTORRENT_ENABLED"
	envQbittorrentHost     = "QBITTORRENT_HOST"
	envQbittorrentPort     = "QBITTORRENT_PORT"
	envQbittorrentUser     = "QBITTORRENT_USER"
	envQbittorrentPass     = "QBITTORRENT_PASS"
	envDelugeEnabled       = "DELUGE_ENABLED"
	envDelugeHost          = "DELUGE_HOST"
	envDelugePort          = "DELUGE_PORT"
	envDelugeUser          = "DELUGE_USER"
	envDelugePass          = "DELUGE_PASS"
)

func GetLogLevel() (int, error) {
	levelStr, present := os.LookupEnv(envLogLevel)
	var logLevel = logging.INFO
	if present {
		switch levelStr {
		case "DEBUG":
			logLevel = logging.DEBUG
		case "INFO":
			logLevel = logging.INFO
		case "WARN":
			logLevel = logging.WARN
		case "ERROR":
			logLevel = logging.ERROR
		default:
			return logging.INFO, fmt.Errorf("env.%s has invalid value %s. Valid values are DEBUG, INFO, WARN, ERROR", envLogLevel, levelStr)
		}
	}
	return logLevel, nil
}

func GetDelaySuccess() (time.Duration, error) {
	return getDuration(envDelaySuccess, 10*time.Minute)
}

func GetDelayError() (time.Duration, error) {
	return getDuration(envDelaySuccess, 5*time.Minute)
}

func getDuration(envVar string, def time.Duration) (time.Duration, error) {
	str, present := os.LookupEnv(envVar)
	if present {
		i, err := strconv.Atoi(str)
		if err != nil || i <= 0 {
			return def, fmt.Errorf("env.%s has invalid value: %s. Valid values are any number of seconds > 0", envVar, str)
		}
		return time.Duration(i) * time.Minute, nil
	}
	return def, nil
}

func getPort(envVar string, def int) (int, error) {
	portStr, present := os.LookupEnv(envVar)
	port := def
	if present {
		i, err := strconv.Atoi(portStr)
		if err != nil || i < 0 || i > 65535 {
			return def, fmt.Errorf("invalid port number found in env.%s: %s", envVar, portStr)
		}
		port = i
	}
	return port, nil
}

func getBool(envVar string, def bool) bool {
	str, present := os.LookupEnv(envVar)
	if present {
		return strings.ToLower(str) == "true"
	}
	return def
}

func GetGluetunClient(httpClient *http.Client, logger logging.Logger) (*gluetun.Client, error) {
	host, present := os.LookupEnv(envGluetunHost)
	if !present {
		host = "localhost"
	}

	port, err := getPort(envGluetunPort, 8000)
	if err != nil {
		return nil, err
	}

	c := gluetun.NewClient(host, port, httpClient, logger)
	c.Log.Info("Client ready %s", c)
	return c, nil
}

func GetTransmissionClient(httpClient *http.Client, logger logging.Logger) (*transmission.Client, error) {
	enabled := getBool(envTransmissionEnabled, false)
	if !enabled {
		logger.Debug("Transmission disabled")
		return nil, nil
	}

	host, present := os.LookupEnv(envTransmissionHost)
	if !present {
		host = "localhost"
	}

	port, err := getPort(envTransmissionPort, 9091)
	if err != nil {
		return nil, err
	}

	user, present := os.LookupEnv(envTransmissionUser)
	if !present {
		user = "admin"
	}

	pass, present := os.LookupEnv(envTransmissionPass)
	if !present {
		pass = "password"
	}

	c := transmission.NewClient(host, port, user, pass, httpClient, logger)
	c.Log.Info("Client ready %s", c)
	return c, nil
}

func GetQbittorrentClient(httpClient *http.Client, logger logging.Logger) (*qbittorrent.Client, error) {
	enabled := getBool(envQbittorrentEnabled, false)
	if !enabled {
		logger.Debug("QBittorrent disabled")
		return nil, nil
	}

	host, present := os.LookupEnv(envQbittorrentHost)
	if !present {
		host = "localhost"
	}

	port, err := getPort(envQbittorrentPort, 8080)
	if err != nil {
		return nil, err
	}

	user, present := os.LookupEnv(envQbittorrentUser)
	if !present {
		user = "admin"
	}

	pass, present := os.LookupEnv(envQbittorrentPass)
	if !present {
		pass = "adminadmin"
	}

	c := qbittorrent.NewClient(host, port, user, pass, httpClient, logger)
	c.Log.Info("Client ready %s", c)
	return c, nil
}

func GetDelugeClient(httpClient *http.Client, logger logging.Logger) (*deluge.Client, error) {
	enabled := getBool(envDelugeEnabled, false)
	if !enabled {
		logger.Debug("Delug disabled")
		return nil, nil
	}

	host, present := os.LookupEnv(envDelugeHost)
	if !present {
		host = "localhost"
	}

	port, err := getPort(envDelugePort, 8112)
	if err != nil {
		return nil, err
	}

	user, present := os.LookupEnv(envDelugeUser)
	if !present {
		user = "admin"
	}

	pass, present := os.LookupEnv(envDelugePass)
	if !present {
		pass = "deluge"
	}

	c := deluge.NewClient(host, port, user, pass, httpClient, logger)
	c.Log.Info("Client ready %s", c)
	return c, nil
}
