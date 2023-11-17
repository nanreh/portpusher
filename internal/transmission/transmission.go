package transmission

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nanreh/portpusher/internal/logging"
)

type Client struct {
	client    *http.Client
	host      string
	port      int
	user      string
	pass      string
	Log       logging.Logger
	authToken string
	portInfo  *portInfo
	sessionId string // returned by Transmission on first http response. https://github.com/transmission/transmission/blob/main/docs/rpc-spec.md#231-csrf-protection
}

// Stringer
func (c *Client) String() string {
	return fmt.Sprintf("host=%s port=%d", c.host, c.port)
}

func NewClient(host string, port int, user string, pass string, httpClient *http.Client, logger logging.Logger) *Client {
	logger = &logging.PrefixLogger{Log: logger, Prefix: "transmission: "}
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
	err := c.init()
	if err != nil {
		return fmt.Errorf("init failed: %v", err)
	}
	c.Log.Debug("init okay sessionId=%s", c.sessionId)

	trPortInfo, err := c.getPortInfo()
	if nil != err {
		return fmt.Errorf("getPortInfo failed: %s", err)
	}
	c.Log.Debug("getPortInfo okay portInfo=%v", trPortInfo)

	if err = c.pushPort(port); nil != err {
		return fmt.Errorf("push failed: %s", err)
	}
	c.Log.Debug("push OK")
	return nil
}

type arguments struct {
	PeerPort       int      `json:"peer-port"`
	PeerPortRandom bool     `json:"peer-port-random-on-start"`
	Fields         []string `json:"fields"`
}
type request struct {
	Arguments arguments `json:"arguments"`
	Method    string    `json:"method"`
	Tag       int       `json:"tag"`
}

type response struct {
	Arguments arguments `json:"arguments"`
	Result    string    `json:"result"`
	Tag       int       `json:"tag"`
}

type portInfo struct {
	PeerPort       int
	PeerPortRandom bool
}

func (c *Client) getUri() string {
	return fmt.Sprintf("http://%s:%d/transmission/rpc", c.host, c.port)
}

func (c *Client) getAuthToken() string {
	authToken := fmt.Sprintf("%s:%s", c.user, c.pass)
	authToken = base64.StdEncoding.EncodeToString([]byte(authToken))
	c.authToken = authToken
	return c.authToken
}

func (c *Client) init() error {
	// make a request to `session-get` to force an HTTP 409 Conflict to get the transmission session id
	body := request{
		Method:    "session-get",
		Arguments: arguments{Fields: []string{"peer-port-random-on-start", "peer-port"}},
	}
	out, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal HTTP body %s", err)
	}
	req, err := c.newRequest(http.MethodPost, c.getUri(), bytes.NewBuffer(out))
	if err != nil {
		return fmt.Errorf("failed to build HTTP request %s", err)
	}

	req.Header.Add("Authorization", "Basic "+c.getAuthToken())

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %s", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusUnauthorized:
		// clear the current session Id, it's invalid
		c.sessionId = ""
	case http.StatusConflict:
		sid := res.Header.Get("X-Transmission-Session-Id")
		if sid == "" {
			return fmt.Errorf("expected session id not received")
		}
		c.sessionId = sid
	default:
		return fmt.Errorf("expected HTTP 409 not received from Transmission, got HTTP %d", res.StatusCode)
	}
	return nil
}

func (c *Client) getPortInfo() (*portInfo, error) {
	// make a request to `session-get` to force an HTTP 409 Conflict to get the transmission session id
	body := request{
		Method:    "session-get",
		Arguments: arguments{Fields: []string{"peer-port-random-on-start", "peer-port"}},
	}
	out, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal HTTP body %s", err)
	}
	c.Log.Debug("getPortInfo request=%v", string(out))

	req, err := c.newRequest(http.MethodPost, c.getUri(), bytes.NewBuffer(out))
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request %s", err)
	}

	req.Header.Add("Authorization", "Basic "+c.getAuthToken())
	req.Header.Add("X-Transmission-Session-Id", c.sessionId)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending HTTP request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request error, got Http %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response %s", err)
	}
	var trResp *response
	err = json.Unmarshal([]byte(data), &trResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal HTTP body: %s", err)
	}
	c.Log.Debug("getPortInfo response=%v", trResp)
	c.portInfo = &portInfo{
		PeerPort:       trResp.Arguments.PeerPort,
		PeerPortRandom: trResp.Arguments.PeerPortRandom,
	}
	return c.portInfo, nil
}

func (c *Client) pushPort(port int) error {
	if c.portInfo.PeerPort == port && !c.portInfo.PeerPortRandom {
		c.Log.Info("Port is correct")
		return nil
	}
	c.Log.Info("Pushing port %d, current port is %d", port, c.portInfo.PeerPort)
	body := request{
		Method: "session-set",
		Arguments: arguments{
			PeerPort:       port,
			PeerPortRandom: false,
		},
	}
	out, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal HTTP body %s", err)
	}
	c.Log.Debug("pushPort response=%v", string(out))
	req, err := c.newRequest(http.MethodPost, c.getUri(), bytes.NewBuffer(out))
	if err != nil {
		return fmt.Errorf("failed to build HTTP request %s", err)
	}

	req.Header.Add("Authorization", "Basic "+c.getAuthToken())
	req.Header.Add("X-Transmission-Session-Id", c.sessionId)

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request error, got Http %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading response %s", err)
	}
	var trResp *response
	err = json.Unmarshal([]byte(data), &trResp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal HTTP body: %s", err)
	}
	c.Log.Debug("pushPort OK response=%v", trResp)
	c.Log.Info("Port pushed")
	return nil
}
