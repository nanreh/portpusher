package deluge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nanreh/portpusher/internal/logging"
)

type Client struct {
	host          string
	port          int
	pass          string
	user          string
	client        *http.Client
	Log           logging.Logger
	nextMessageId int
}

// Stringer
func (c *Client) String() string {
	return fmt.Sprintf("host=%s port=%d", c.host, c.port)
}

func NewClient(host string, port int, user string, pass string, httpClient *http.Client, logger logging.Logger) *Client {
	logger = &logging.PrefixLogger{Log: logger, Prefix: "deluge: "}
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
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", fmt.Sprintf("http://%s:%d", c.host, c.port))
	req.Header.Add("Referer", fmt.Sprintf("http://%s:%d", c.host, c.port))
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
	loginRes, err := c.login()
	if err != nil {
		return err
	}
	c.Log.Debug("login OK response=%v", loginRes)

	webConnectedRes, err := c.webConnected()
	if err != nil {
		return err
	}

	if !webConnectedRes.ResultBool {
		c.Log.Debug("Deluge disconnected")

		hosts, err := c.getHosts()
		if err != nil {
			return err
		}
		c.Log.Debug("Hosts: %v", hosts.ResultHosts)

		if len(hosts.ResultHosts) == 0 {
			return fmt.Errorf("no hosts to connect to")
		}

		connected := false
		for _, h := range hosts.ResultHosts {
			res, err := c.getHostStatus(h.Id)
			if err != nil {
				return err
			}
			hostStatus := res.ResultHostStatus
			if hostStatus.Status == "Online" {
				_, err = c.webConnect(h.Id)
				if err != nil {
					return err
				}
				c.Log.Debug("connected to host %s", h.Id)
				connected = true
			}
		}
		if !connected {
			return fmt.Errorf("no online deluge hosts found")
		}
	} else {
		c.Log.Debug("Deluge is connected")
	}

	config, err := c.getConfig()
	if err != nil {
		return err
	}
	c.Log.Debug("getConfig OK %v", config)

	if port == config.ListenPorts[0] {
		c.Log.Info("Port is correct")
		return nil
	}

	c.Log.Info("Pushing port %d, current port is %d", port, config.ListenPorts[0])
	if err = c.setConfig(port); err != nil {
		return err
	}
	c.Log.Info("Port pushed")
	return nil
}

func (c *Client) nextId() int {
	c.nextMessageId = c.nextMessageId + 1
	return c.nextMessageId
}

type request struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
	Id     int           `json:"id"`
}

type response struct {
	Error            *errorResponse
	Id               int
	ResultBool       bool
	ResultStrings    []string
	ResultHosts      []host
	ResultHostStatus hostStatus
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type genericResponse struct {
	Error  *errorResponse `json:"error"`
	Id     int            `json:"id"`
	Result interface{}    `json:"result"`
}

type config struct {
	RandomPort  bool  `json:"random_port"`
	ListenPorts []int `json:"listen_ports"`
}

type getConfigResponse struct {
	Error  *errorResponse `json:"error"`
	Id     int            `json:"id"`
	Result config         `json:"result"`
}

type host struct {
	Id       string
	Addr     string
	Port     float64
	Hostname string
}

func (h *host) From(arry []interface{}) {
	if len(arry) >= 4 {
		h.Id = arry[0].(string)
		h.Addr = arry[1].(string)
		h.Port = arry[2].(float64)
		h.Hostname = arry[3].(string)
	}
}

// sample value: ["a92774accdd846f48179a892494625cc", "Online", "2.1.1"],
type hostStatus struct {
	Id      string
	Status  string
	Version string
}

func (s *hostStatus) From(arry []interface{}) {
	if len(arry) >= 3 {
		s.Id = arry[0].(string)
		s.Status = arry[1].(string)
		s.Version = arry[2].(string)
	}
}

func boolHandler(resp *genericResponse, err error) (*response, error) {
	if err != nil {
		return nil, err
	}
	r := &response{
		Error: resp.Error,
		Id:    resp.Id,
	}
	switch t := resp.Result.(type) {
	case bool:
		r.ResultBool = t
	default:
		return nil, fmt.Errorf("expected boolean but found %T %v", t, t)
	}
	return r, nil
}

func stringsSliceHandler(resp *genericResponse, err error) (*response, error) {
	if err != nil {
		return nil, err
	}
	r := &response{
		Error: resp.Error,
		Id:    resp.Id,
	}
	switch t := resp.Result.(type) {
	case []interface{}:
		strings := make([]string, len(t))
		for i, v := range t {
			strings[i] = v.(string)
		}
		r.ResultStrings = strings
	default:
		return nil, fmt.Errorf("expected string slice but found %T %v", t, t)
	}
	return r, nil
}

func hostsSliceHandler(resp *genericResponse, err error) (*response, error) {
	if err != nil {
		return nil, err
	}
	r := &response{
		Error: resp.Error,
		Id:    resp.Id,
	}
	switch t := resp.Result.(type) {
	case []interface{}:
		hosts := make([]host, len(t))
		for n, i := range t {
			var h = host{}
			h.From(i.([]interface{}))
			hosts[n] = h
		}
		r.ResultHosts = hosts
	default:
		return nil, fmt.Errorf("expected []hosts but found %T %v", t, t)
	}
	return r, nil
}

func hostStatusHandler(resp *genericResponse, err error) (*response, error) {
	if err != nil {
		return nil, err
	}
	r := &response{
		Error: resp.Error,
		Id:    resp.Id,
	}
	switch t := resp.Result.(type) {
	case []interface{}:
		var h = hostStatus{}
		h.From(t)
		r.ResultHostStatus = h
	default:
		return nil, fmt.Errorf("expected []hostStatus but found %T %v", t, t)
	}
	return r, nil
}

func (c *Client) delugeRequest(method string, params []interface{}) (*genericResponse, error) {
	body := request{
		Method: method,
		Params: params,
		Id:     c.nextId(),
	}
	out, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request %s", err)
	}
	c.Log.Debug("%s request=%v", method, string(out))

	uri := fmt.Sprintf("http://%s:%d/json", c.host, c.port)
	req, err := c.newRequest(http.MethodPost, uri, bytes.NewBuffer(out))
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request %s", err)
	}

	c.Log.Debug("request=%v", req)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s failed: %s", method, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s failed, got HTTP %d", method, res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response %s", err)
	}

	var resp *genericResponse
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}
	if nil != resp.Error {
		return nil, fmt.Errorf("error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return resp, nil
}

// Login into Deluge Web
//
// sample request:
//
//	{"method":"auth.login","params":["deluge"],"id":46}
//
// sample response:
//
//	{"result": true, "error": null, "id": 46}
func (c *Client) login() (*response, error) {
	return boolHandler(c.delugeRequest("auth.login", []interface{}{c.pass}))
}

// Checks if Deluge Web is connected to daemon
//
// sample request:
//
//	{"method":"web.connected","params":[],"id":52}
//
// sample response:
//
//	{"result": false, "error": null, "id": 52}
func (c *Client) webConnected() (*response, error) {
	return boolHandler(c.delugeRequest("web.connected", []interface{}{}))
}

// Connects to a daemon server
//
// sample request:
//
//	{"method":"web.connect","params":["a92774accdd846f48179a892494625cc"],"id":16}
//
// sample response
//
//	{
//		"result": ["core.add_torrent_file", "core.add_torrent_file_async", "core.add_torrent_files", "core.add_torrent_magnet", "core.add_torrent_url", "core.connect_peer", "core.create_account", "core.create_torrent", "core.disable_plugin", "core.enable_plugin", "core.force_reannounce", "core.force_recheck", "core.get_auth_levels_mappings", "core.get_available_plugins", "core.get_completion_paths", "core.get_config", "core.get_config_value", "core.get_config_values", "core.get_enabled_plugins", "core.get_external_ip", "core.get_filter_tree", "core.get_free_space", "core.get_known_accounts", "core.get_libtorrent_version", "core.get_listen_port", "core.get_magnet_uri", "core.get_path_size", "core.get_proxy", "core.get_session_state", "core.get_session_status", "core.get_torrent_status", "core.get_torrents_status", "core.glob", "core.is_session_paused", "core.move_storage", "core.pause_session", "core.pause_torrent", "core.pause_torrents", "core.prefetch_magnet_metadata", "core.queue_bottom", "core.queue_down", "core.queue_top", "core.queue_up", "core.remove_account", "core.remove_torrent", "core.remove_torrents", "core.rename_files", "core.rename_folder", "core.rescan_plugins", "core.resume_session", "core.resume_torrent", "core.resume_torrents", "core.set_config", "core.set_torrent_auto_managed", "core.set_torrent_file_priorities", "core.set_torrent_max_connections", "core.set_torrent_max_download_speed", "core.set_torrent_max_upload_slots", "core.set_torrent_max_upload_speed", "core.set_torrent_move_completed", "core.set_torrent_move_completed_path", "core.set_torrent_options", "core.set_torrent_prioritize_first_last", "core.set_torrent_remove_at_ratio", "core.set_torrent_stop_at_ratio", "core.set_torrent_stop_ratio", "core.set_torrent_trackers", "core.test_listen_port", "core.update_account", "core.upload_plugin", "daemon.authorized_call", "daemon.get_method_list", "daemon.get_version", "daemon.shutdown"],
//		"error": null,
//		"id": 16
//	}
func (c *Client) webConnect(hostname string) (*response, error) {
	return stringsSliceHandler(c.delugeRequest("web.connect", []interface{}{hostname}))
}

// Gets list of hosts
// sample request:
//
//	{ "method": "web.get_hosts", "params": [], "id": 8 }
//
// sample response:
//
//	{
//		"result": [
//			["a92774accdd846f48179a892494625cc", "127.0.0.1", 58846, "localclient"]
//		],
//		"error": null,
//		"id": 8
//	}
func (c *Client) getHosts() (*response, error) {
	return hostsSliceHandler(c.delugeRequest("web.get_hosts", []interface{}{}))
}

// Gets status of a single host given its id

// sample request:
//
//	{"method":"web.get_host_status","params":["a92774accdd846f48179a892494625cc"],"id":15}
//
// sample response:
//
//	 {
//		 "result": ["a92774accdd846f48179a892494625cc", "Online", "2.1.1"],
//		 "error": null,
//		 "id": 15
//	 }
func (c *Client) getHostStatus(hostId string) (*response, error) {
	return hostStatusHandler(c.delugeRequest("web.get_host_status", []interface{}{hostId}))
}

func (c *Client) getConfig() (*config, error) {
	body := request{
		Method: "core.get_config",
		Params: []interface{}{},
		Id:     c.nextId(),
	}
	out, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request %s", err)
	}
	c.Log.Debug("getConfig request=%v", string(out))

	uri := fmt.Sprintf("http://%s:%d/json", c.host, c.port)
	req, err := c.newRequest(http.MethodPost, uri, bytes.NewBuffer(out))
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request %s", err)
	}

	c.Log.Debug("request=%v", req)
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

	var resp *getConfigResponse
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}
	if nil != resp.Error {
		return nil, fmt.Errorf("error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return &resp.Result, nil
}

func (c *Client) setConfig(port int) error {
	paramMap := make(map[string]interface{})
	paramMap["listen_ports"] = []int{port, port}
	paramMap["random_port"] = false
	body := request{
		Method: "core.set_config",
		Params: []interface{}{paramMap},
		Id:     c.nextId(),
	}
	out, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to build HTTP request %s", err)
	}
	c.Log.Debug("setConfig request=%v", string(out))

	uri := fmt.Sprintf("http://%s:%d/json", c.host, c.port)
	req, err := c.newRequest(http.MethodPost, uri, bytes.NewBuffer(out))
	if err != nil {
		return fmt.Errorf("failed to build HTTP request %s", err)
	}

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
	c.Log.Debug("setConfig response=%v", string(data))

	var resp *genericResponse
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return fmt.Errorf("could not unmarshal json: %s", err)
	}

	if nil != resp.Error {
		return fmt.Errorf("error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	c.Log.Debug("setConfig OK resp=%v", resp)
	return nil
}
