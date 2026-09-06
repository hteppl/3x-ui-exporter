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

// v3 API response envelopes. Only the fields the exporter reads are declared;
// unknown keys (settings, streamSettings, sniffing — nested JSON objects in v3)
// are ignored. This is exactly why the old string-typed inbound schema could
// not decode a v3 /inbounds/list response.
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
			Version string `json:"version"`
		} `json:"xray"`
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

var authCache struct {
	Cookie    http.Cookie
	CSRFToken string
	ExpiresAt time.Time
	sync.Mutex
}

// fetchCSRFToken retrieves the session CSRF token required by 3X-UI v3.0+.
// The token is bound to the session cookie the panel sets on this response, so
// both are returned and must be carried into the subsequent /login request.
// Older panels lack this endpoint; on any failure it returns ("", nil), which
// GetAuthToken now treats as a hard authentication failure (3X-UI v3.0+ is required).
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
		return &authCache.Cookie, nil
	}

	// 3X-UI v3.0+ requires a CSRF token bound to the session cookie that
	// /csrf-token sets. Without it the panel returns HTTP 403 on /login.
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
		// The token is validated against the session /csrf-token created, so
		// that session cookie must ride along with the login request.
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
		return nil, fmt.Errorf("authentication failed: HTTP 403 (CSRF token rejected?)")
	}

	var loginResp struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return nil, err
	}

	if !loginResp.Success {
		return nil, fmt.Errorf("authentication failed: %s", loginResp.Msg)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication: code %s", resp.Status)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" {
			authCache.Cookie = *cookie
			authCache.ExpiresAt = time.Now().Add(time.Minute * 59)
		}
	}

	if authCache.Cookie.Name == "" {
		return nil, fmt.Errorf("no cookies found in auth response")
	}

	authCache.CSRFToken = csrfToken

	return &authCache.Cookie, nil
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

func (a *APIClient) FetchServerStatus(cookie *http.Cookie) error {
	// Clear old version metric to avoid accumulating obsolete label values
	metrics.XrayVersion.Reset()

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

	// XRay metrics — parsing preserved byte-for-byte from the pre-v3 code so the
	// gauge value for a given version string does not change.
	xrayVersion := strings.ReplaceAll(response.Obj.Xray.Version, ".", "")
	num, _ := strconv.ParseFloat(xrayVersion, 64)
	metrics.XrayVersion.WithLabelValues(response.Obj.Xray.Version).Set(num)

	// Panel metrics
	metrics.PanelThreads.Set(float64(response.Obj.AppStats.Threads))
	metrics.PanelMemory.Set(float64(response.Obj.AppStats.Mem))
	metrics.PanelUptime.Set(float64(response.Obj.AppStats.Uptime))

	return nil
}

func (a *APIClient) FetchInboundsList(cookie *http.Cookie) error {
	// Clear old metric values to avoid exposing stale data from previous
	// updates. Resetting ensures obsolete label combinations are removed
	// before setting new values.
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
			// Top N by Upload
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

			// Top N by Download
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

	// 3X-UI v3.0+ validates the CSRF token on API requests. Some panels reject
	// even safe methods without it, so it is attached to every request. Empty
	// only before the first successful login, where it is a harmless no-op.
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
