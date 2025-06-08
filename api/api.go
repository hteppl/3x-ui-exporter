package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/digilolnet/client3xui"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"x-ui-exporter/config"
	"x-ui-exporter/metrics"
)

// API logic partially was taken from the client3xui module
// https://github.com/digilolnet/client3xui

var cookieCache struct {
	Cookie    http.Cookie
	ExpiresAt time.Time
	sync.Mutex
}

func createHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
}

func GetAuthToken() (*http.Cookie, error) {
	cookieCache.Lock()
	defer cookieCache.Unlock()

	remainingTime := time.Until(cookieCache.ExpiresAt).Minutes()
	if cookieCache.Cookie.Name != "" && remainingTime > 0 {
		log.Printf("Login cookies will expire in %.2f minutes", remainingTime)
		return &cookieCache.Cookie, nil
	}

	path := config.CLIConfig.BaseURL + "/login"
	data := url.Values{
		"username": {config.CLIConfig.ApiUsername},
		"password": {config.CLIConfig.ApiPassword},
	}

	req, err := http.NewRequest("POST", path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := createHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var loginResp struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return nil, err
	}

	if !loginResp.Success {
		return nil, errors.New(loginResp.Msg)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("authentication failed")
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" {
			cookieCache.Cookie = *cookie
			cookieCache.ExpiresAt = time.Now().Add(time.Minute * 59)
		}
	}

	if cookieCache.Cookie.Name == "" {
		return nil, errors.New("no cookies found in auth response")
	}

	return &cookieCache.Cookie, nil
}

func FetchOnlineUsersCount(cookie *http.Cookie) {
	body, err := sendRequest("/panel/inbound/onlines", http.MethodPost, cookie)
	if err != nil {
		log.Println("Error making request for inbound onlines:", err)
		return
	}

	var response client3xui.ApiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Println("Error unmarshaling response:", err)
		return
	}

	var arr []json.RawMessage
	err = json.Unmarshal(response.Obj, &arr)
	if err != nil {
		log.Println("Error converting Obj as array:", err)
		return
	}

	metrics.OnlineUsersCount.Set(float64(len(arr)))
}

func FetchServerStatus(cookie *http.Cookie) {
	body, err := sendRequest("/server/status", http.MethodPost, cookie)
	if err != nil {
		log.Println("Error making request for system stats:", err)
		return
	}

	var response client3xui.ServerStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Println("Error unmarshaling response:", err)
		return
	}

	// XRay metrics
	xrayVersion := strings.ReplaceAll(response.Obj.Xray.Version, ".", "")
	num, err := strconv.ParseFloat(xrayVersion, 64)
	if err != nil {
		log.Println("Error converting xrayVersion:", err)
		metrics.XrayVersion.WithLabelValues(response.Obj.Xray.Version).Set(0)
	} else {
		metrics.XrayVersion.WithLabelValues(response.Obj.Xray.Version).Set(num)
	}

	// Panel metrics
	metrics.PanelThreads.Set(float64(response.Obj.AppStats.Threads))
	metrics.PanelMemory.Set(float64(response.Obj.AppStats.Mem))
	metrics.PanelUptime.Set(float64(response.Obj.AppStats.Uptime))
}

func FetchInboundsList(cookie *http.Cookie) {
	body, err := sendRequest("/panel/api/inbounds/list", http.MethodGet, cookie)
	if err != nil {
		log.Println("Error making request for inbounds list:", err)
		return
	}

	var response client3xui.GetInboundsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Println("Error unmarshaling response:", err)
		return
	}

	for _, inbound := range response.Obj {
		metrics.InboundUp.WithLabelValues(
			strconv.Itoa(inbound.ID), inbound.Remark,
		).Set(float64(inbound.Up))

		metrics.InboundDown.WithLabelValues(
			strconv.Itoa(inbound.ID), inbound.Remark,
		).Set(float64(inbound.Down))

		for _, client := range inbound.ClientStats {
			metrics.ClientUp.WithLabelValues(
				strconv.Itoa(client.ID), client.Email,
			).Set(float64(client.Up))

			metrics.ClientDown.WithLabelValues(
				strconv.Itoa(client.ID), client.Email,
			).Set(float64(client.Down))
		}
	}
}

func createRequest(method, path string, cookie *http.Cookie) (*http.Request, error) {
	requestUrl := fmt.Sprintf("%s%s", config.CLIConfig.BaseURL, path)

	req, err := http.NewRequest(method, requestUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.AddCookie(cookie)
	return req, nil
}

func sendRequest(path, method string, cookie *http.Cookie) ([]byte, error) {
	req, err := createRequest(method, path, cookie)
	if err != nil {
		return nil, err
	}

	resp, err := createHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
