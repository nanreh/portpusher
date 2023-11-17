package qbittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nanreh/portpusher/internal/logging"
)

type Client struct {
	host   string
	port   int
	user   string
	pass   string
	client *http.Client
	Log    logging.Logger
}

// Stringer
func (c *Client) String() string {
	return fmt.Sprintf("host=%s port=%d", c.host, c.port)
}

func NewClient(host string, port int, user string, pass string, httpClient *http.Client, logger logging.Logger) *Client {
	logger = &logging.PrefixLogger{Log: logger, Prefix: "qbittorrent: "}
	return &Client{
		host:   host,
		port:   port,
		user:   user,
		pass:   pass,
		client: httpClient,
		Log:    logger,
	}
}

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Port Pusher")
	return req, nil
}

// PortPusher
func (c *Client) Push(port int) error {
	if err := c.doPush(port); err != nil {
		c.Log.Error("push error: %v", err)
		return err
	}
	return nil
}

func (c *Client) doPush(port int) error {
	if err := c.login(); err != nil {
		return err
	}
	c.Log.Debug("login OK")

	prefs, err := c.getPreferences()
	if err != nil {
		return err
	}
	c.Log.Debug("getPreferences OK %v", prefs)

	if prefs.Port == port && !prefs.PortRandom {
		// nothing to do
		c.Log.Info("Port is correct")
	} else {
		c.Log.Info("Pushing port %d, current port is %d", port, prefs.Port)
		err = c.setPreferences(&preferences{Port: port, PortRandom: false})
		if err != nil {
			return err
		}
		c.Log.Info("Port pushed")
	}
	return nil
}

type preferences struct {
	Port       int  `json:"listen_port"`
	PortRandom bool `json:"random_port"`
}

func (c *Client) login() error {
	data := url.Values{}
	data.Set("username", c.user)
	data.Set("password", c.pass)
	baseUri := fmt.Sprintf("http://%s:%d", c.host, c.port)
	uri := fmt.Sprintf("%s/api/v2/auth/login", baseUri)
	req, err := c.newRequest(http.MethodPost, uri, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to build HTTP request %s", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Referrer", baseUri) // see https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#login

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request error, got HTTP %d", res.StatusCode)
	}
	c.Log.Debug("login response %v", res)
	return nil
}

func (c *Client) getPreferences() (*preferences, error) {
	baseUri := fmt.Sprintf("http://%s:%d", c.host, c.port)
	uri := fmt.Sprintf("%s/api/v2/app/preferences", baseUri)
	req, err := c.newRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request %s", err)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending HTTP request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request error, got HTTP %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response %s", err)
	}
	var prefs *preferences
	err = json.Unmarshal([]byte(data), &prefs)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}
	return prefs, nil
}

func (c *Client) setPreferences(prefs *preferences) error {
	prefsJson, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("failed to build HTTP request %s", err)
	}

	data := url.Values{}
	data.Set("json", string(prefsJson))
	baseUri := fmt.Sprintf("http://%s:%d", c.host, c.port)
	uri := fmt.Sprintf("%s/api/v2/app/setPreferences", baseUri)
	req, err := c.newRequest(http.MethodPost, uri, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to build HTTP request %s", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request error, got HTTP %d", res.StatusCode)
	}

	return nil
}
