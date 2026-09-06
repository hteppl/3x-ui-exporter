package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"x-ui-exporter/metrics"
)

type APIConfig struct {
	BaseURL            string
	ApiUsername        string
	ApiPassword        string
	InsecureSkipVerify bool
	ClientsBytesRows   int
}

type APIClient struct {
	config     APIConfig
	httpClient *http.Client
}

// v3 API envelopes; only the fields the exporter reads are declared.
type onlinesResponse struct {
	Success bool     `json:"success"`
	Msg     string   `json:"msg"`
	Obj     []string `json:"obj"`
}

type serverStatusResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Obj     struct {
		Xray struct {
			Version  string `json:"version"`
			State    string `json:"state"`
			ErrorMsg string `json:"errorMsg"`
		} `json:"xray"`
		PanelVersion string `json:"panelVersion"`
		AmneziaWG *struct {
			Running bool `json:"running"`
		} `json:"amneziawg"`
		AppStats struct {
			Threads uint32 `json:"threads"`
			Mem     uint64 `json:"mem"`
			Uptime  uint64 `json:"uptime"`
		} `json:"appStats"`
	} `json:"obj"`
}

type inboundsResponse struct {
	Success bool      `json:"success"`
	Msg     string    `json:"msg"`
	Obj     []inbound `json:"obj"`
}

type inbound struct {
	ID          int          `json:"id"`
	Up          int64        `json:"up"`
	Down        int64        `json:"down"`
	Remark      string       `json:"remark"`
	ClientStats []clientStat `json:"clientStats"`
}

type clientStat struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Up    int64  `json:"up"`
	Down  int64  `json:"down"`
}

func NewAPIClient(cfg APIConfig) *APIClient {
	return &APIClient{
		config: cfg,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: cfg.InsecureSkipVerify,
				},
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: 30 * time.Second,
		},
	}
}

// 3X-UI blocks an IP/username pair for 15 minutes after 5 failed logins.
const (
	authRetryBaseDelay = 30 * time.Second
	authRetryMaxDelay  = 15 * time.Minute
)

var authCache struct {
	Cookie      http.Cookie
	CSRFToken   string
	ExpiresAt   time.Time
	Failures    int
	NextRetryAt time.Time
	sync.Mutex
}

// registerAuthFailure schedules the next permitted attempt. Caller holds the lock.
func registerAuthFailure() time.Duration {
	authCache.Failures++

	delay := authRetryBaseDelay
	for i := 1; i < authCache.Failures && delay < authRetryMaxDelay; i++ {
		delay *= 2
	}
	if delay > authRetryMaxDelay {
		delay = authRetryMaxDelay
	}

	authCache.NextRetryAt = time.Now().Add(delay)
	return delay
}

// resetAuthFailures clears the backoff. Caller holds the lock.
func resetAuthFailures() {
	authCache.Failures = 0
	authCache.NextRetryAt = time.Time{}
}

// fetchCSRFToken returns the v3.0+ CSRF token and the session cookie it is bound
// to; both must ride along on /login. Returns ("", nil) on any failure.
func (a *APIClient) fetchCSRFToken() (string, *http.Cookie) {
	req, err := http.NewRequest(http.MethodGet, a.config.BaseURL+"/csrf-token", nil)
	if err != nil {
		return "", nil
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	var csrfResp struct {
		Success bool   `json:"success"`
		Obj     string `json:"obj"`
	}
	if err := json.Unmarshal(body, &csrfResp); err != nil || !csrfResp.Success || csrfResp.Obj == "" {
		return "", nil
	}

	var sessionCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" {
			sessionCookie = cookie
		}
	}

	return csrfResp.Obj, sessionCookie
}

func (a *APIClient) GetAuthToken() (*http.Cookie, error) {
	authCache.Lock()
	defer authCache.Unlock()

	remainingTime := time.Until(authCache.ExpiresAt).Minutes()
	if authCache.Cookie.Name != "" && remainingTime > 0 {
		// Copy: the caller reads this after the lock is dropped, and a later
		// login overwrites authCache.Cookie in place.
		cookie := authCache.Cookie
		return &cookie, nil
	}

	// A bad credential must not trip the panel's login limiter.
	if wait := time.Until(authCache.NextRetryAt); wait > 0 {
		return nil, fmt.Errorf(
			"authentication backoff after %d failed attempt(s); retrying in %s",
			authCache.Failures, wait.Round(time.Second),
		)
	}

	// Without the CSRF token the panel returns HTTP 403 on /login.
	csrfToken, csrfCookie := a.fetchCSRFToken()
	if csrfToken == "" {
		return nil, fmt.Errorf("could not obtain CSRF token; 3X-UI v3.0+ required")
	}

	path := a.config.BaseURL + "/login"
	data := url.Values{
		"username": {a.config.ApiUsername},
		"password": {a.config.ApiPassword},
	}

	req, err := http.NewRequest("POST", path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrfToken)
	if csrfCookie != nil {
		// The token is validated against the session that minted it.
		req.AddCookie(csrfCookie)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusForbidden {
		delay := registerAuthFailure()
		return nil, fmt.Errorf(
			"authentication failed: HTTP 403 (CSRF token rejected?); backing off %s", delay,
		)
	}

	var loginResp struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		delay := registerAuthFailure()
		return nil, fmt.Errorf("authentication: unparseable login response: %w; backing off %s", err, delay)
	}

	if !loginResp.Success {
		delay := registerAuthFailure()
		return nil, fmt.Errorf("authentication failed: %s; backing off %s", loginResp.Msg, delay)
	}

	if resp.StatusCode != http.StatusOK {
		delay := registerAuthFailure()
		return nil, fmt.Errorf("authentication: code %s; backing off %s", resp.Status, delay)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" {
			authCache.Cookie = *cookie
			authCache.ExpiresAt = time.Now().Add(time.Minute * 59)
		}
	}

	if authCache.Cookie.Name == "" {
		delay := registerAuthFailure()
		return nil, fmt.Errorf("no session cookie in auth response; backing off %s", delay)
	}

	authCache.CSRFToken = csrfToken
	resetAuthFailures()

	cookie := authCache.Cookie
	return &cookie, nil
}

func (a *APIClient) FetchOnlineUsersCount(cookie *http.Cookie) error {
	body, err := a.sendRequest("/panel/api/clients/onlines", http.MethodPost, cookie)
	if err != nil {
		return fmt.Errorf("onlines: %w", err)
	}

	var response onlinesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("onlines: %s", response.Msg)
	}

	metrics.OnlineUsersCount.Set(float64(len(response.Obj)))

	return nil
}

func (a *APIClient) FetchServerStatus(cookie *http.Cookie) (err error) {
	// Drop obsolete label values.
	metrics.XrayVersion.Reset()
	metrics.PanelVersion.Reset()
	metrics.XrayState.Reset()
	metrics.AmneziaWGUp.Reset()

	// The plain gauges below are not label-bearing, so Reset() cannot clear
	// them. Without this an unreachable panel would leave x_ui_xray_up stuck
	// at 1 while its companion series disappear, and the alert that METRICS.md
	// recommends would never fire.
	defer func() {
		if err != nil {
			metrics.XrayUp.Set(0)
		}
	}()

	body, err := a.sendRequest("/panel/api/server/status", http.MethodGet, cookie)
	if err != nil {
		return fmt.Errorf("server status: %w", err)
	}

	var response serverStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("server status: %s", response.Msg)
	}

	// Parsing preserved from pre-v3 so the gauge value per version is unchanged.
	xrayVersion := strings.ReplaceAll(response.Obj.Xray.Version, ".", "")
	num, _ := strconv.ParseFloat(xrayVersion, 64)
	metrics.XrayVersion.WithLabelValues(response.Obj.Xray.Version).Set(num)

	if response.Obj.PanelVersion != "" {
		metrics.PanelVersion.WithLabelValues(response.Obj.PanelVersion).Set(1)
	}

	// Panel reports "running", "stop" or "error"; only "running" is up.
	metrics.XrayState.WithLabelValues(response.Obj.Xray.State, response.Obj.Xray.ErrorMsg).Set(1)
	if response.Obj.Xray.State == "running" {
		metrics.XrayUp.Set(1)
	} else {
		metrics.XrayUp.Set(0)
	}

	// Absent on panels built without AmneziaWG; leave the metric unset there
	// rather than publishing a permanent 0.
	if response.Obj.AmneziaWG != nil {
		if response.Obj.AmneziaWG.Running {
			metrics.AmneziaWGUp.WithLabelValues().Set(1)
		} else {
			metrics.AmneziaWGUp.WithLabelValues().Set(0)
		}
	}

	metrics.PanelGoroutines.Set(float64(response.Obj.AppStats.Threads))
	metrics.PanelMemoryBytes.Set(float64(response.Obj.AppStats.Mem))
	metrics.XrayUptimeSeconds.Set(float64(response.Obj.AppStats.Uptime))

	return nil
}

func (a *APIClient) FetchInboundsList(cookie *http.Cookie) error {
	// Drop obsolete label combinations before setting new values.
	metrics.InboundUp.Reset()
	metrics.InboundDown.Reset()
	metrics.ClientUp.Reset()
	metrics.ClientDown.Reset()

	body, err := a.sendRequest("/panel/api/inbounds/list", http.MethodGet, cookie)
	if err != nil {
		return fmt.Errorf("inbounds list: %w", err)
	}

	var response inboundsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("inbounds list: %s", response.Msg)
	}

	for _, inb := range response.Obj {
		iid := strconv.Itoa(inb.ID)
		metrics.InboundUp.WithLabelValues(iid, inb.Remark).Set(float64(inb.Up))
		metrics.InboundDown.WithLabelValues(iid, inb.Remark).Set(float64(inb.Down))

		n := a.config.ClientsBytesRows
		if n == 0 {
			for _, client := range inb.ClientStats {
				cid := strconv.Itoa(client.ID)
				metrics.ClientUp.WithLabelValues(cid, client.Email).Set(float64(client.Up))
				metrics.ClientDown.WithLabelValues(cid, client.Email).Set(float64(client.Down))
			}
		} else {
			sortedUp := make([]clientStat, len(inb.ClientStats))
			copy(sortedUp, inb.ClientStats)
			sort.Slice(sortedUp, func(i, j int) bool {
				return sortedUp[i].Up > sortedUp[j].Up
			})
			for i := 0; i < n && i < len(sortedUp); i++ {
				client := sortedUp[i]
				metrics.ClientUp.WithLabelValues(
					strconv.Itoa(client.ID), client.Email,
				).Set(float64(client.Up))
			}

			sortedDown := make([]clientStat, len(inb.ClientStats))
			copy(sortedDown, inb.ClientStats)
			sort.Slice(sortedDown, func(i, j int) bool {
				return sortedDown[i].Down > sortedDown[j].Down
			})
			for i := 0; i < n && i < len(sortedDown); i++ {
				client := sortedDown[i]
				metrics.ClientDown.WithLabelValues(
					strconv.Itoa(client.ID), client.Email,
				).Set(float64(client.Down))
			}
		}
	}

	return nil
}

func (a *APIClient) createRequest(method, path string, cookie *http.Cookie) (*http.Request, error) {
	requestUrl := fmt.Sprintf("%s%s", a.config.BaseURL, path)

	req, err := http.NewRequest(method, requestUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.AddCookie(cookie)

	// Some panels reject even safe methods without the token; empty is a no-op.
	authCache.Lock()
	token := authCache.CSRFToken
	authCache.Unlock()
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}

	return req, nil
}

func (a *APIClient) sendRequest(path, method string, cookie *http.Cookie) ([]byte, error) {
	req, err := a.createRequest(method, path, cookie)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return io.ReadAll(resp.Body)
}
