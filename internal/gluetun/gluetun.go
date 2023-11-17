package gluetun

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nanreh/portpusher/internal/logging"
)

type Client struct {
	host   string
	port   int
	client *http.Client
	Log    logging.Logger
}

// Stringer
func (c *Client) String() string {
	return fmt.Sprintf("host=%s port=%d", c.host, c.port)
}

type gtStatusResp struct {
	Status string `json:"status"`
}
type gtPortResp struct {
	Port int `json:"port"`
}

func NewClient(host string, port int, httpClient *http.Client, logger logging.Logger) *Client {
	logger = &logging.PrefixLogger{Log: logger, Prefix: "gluetun: "}
	return &Client{
		host:   host,
		port:   port,
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

// Pull the current forwarded port from Gluetun.
// Makes two calls to Gluetun: one to verify that it's running and a second to fetch the forwarded port.
func (c Client) PullPort() (int, error) {
	port, err := c.doPullPort()
	if err != nil {
		c.Log.Error("pull port error: %v", err)
		return port, err
	}
	return port, err
}

func (c Client) doPullPort() (int, error) {
	// check that gtun is connected
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("http://%s:%d/v1/openvpn/status", c.host, c.port), nil)
	if err != nil {
		return -1, fmt.Errorf("failed to build HTTP request %s", err)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return -1, fmt.Errorf("failed to fetch gluetun status: %s", err)
	}
	if res.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("failed to fetch gluetun status. HTTP status: %d", res.StatusCode)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return -1, fmt.Errorf("error reading response %s", err)
	}
	c.Log.Debug("status response: %s", string(data))

	var statusResp *gtStatusResp
	err = json.Unmarshal([]byte(data), &statusResp)
	if err != nil {
		return -1, fmt.Errorf("could not unmarshal json: %s", err)
	}

	if statusResp.Status != "running" {
		return -1, fmt.Errorf("status is %s, cannot fetch forwarded port", statusResp.Status)
	}

	// fetch the forwarded port
	req, err = c.newRequest(http.MethodGet, fmt.Sprintf("http://%s:%d/v1/openvpn/portforwarded", c.host, c.port), nil)
	if err != nil {
		return -1, fmt.Errorf("failed to build HTTP request %s", err)
	}
	res, err = c.client.Do(req)
	if err != nil {
		return -1, fmt.Errorf("failed to fetch forwarded port: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("failed to fetch forwarded port. HTTP status: %d", res.StatusCode)
	}

	defer res.Body.Close()
	data, err = io.ReadAll(res.Body)
	if err != nil {
		return -1, fmt.Errorf("error reading response %s", err)
	}
	c.Log.Debug("portforwarded response: %s", string(data))

	var portResp *gtPortResp
	err = json.Unmarshal([]byte(data), &portResp)
	if err != nil {
		return -1, fmt.Errorf("could not unmarshal json: %s", err)
	}

	// Gluetun may respond with 0 if port forwarding is not available or if it disconnects
	if portResp.Port == 0 {
		return -1, fmt.Errorf("gluetun responded with port 0")
	}

	c.Log.Info("Forwarded port is %d", portResp.Port)

	return portResp.Port, nil
}
