package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"x-ui-exporter/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// newTestClient points an APIClient at ts and returns a dummy session cookie.
func newTestClient(ts *httptest.Server) (*APIClient, *http.Cookie) {
	c := NewAPIClient(APIConfig{BaseURL: ts.URL})
	return c, &http.Cookie{Name: "3x-ui", Value: "test-session"}
}

func TestFetchOnlineUsersCount(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/panel/api/clients/onlines", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		_, _ = w.Write([]byte(`{"success":true,"obj":["alice","bob","carol"]}`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	c, cookie := newTestClient(ts)
	if err := c.FetchOnlineUsersCount(cookie); err != nil {
		t.Fatalf("FetchOnlineUsersCount: %v", err)
	}
	if got := testutil.ToFloat64(metrics.OnlineUsersCount); got != 3 {
		t.Errorf("OnlineUsersCount = %v, want 3", got)
	}
}

func TestFetchOnlineUsersCountFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/panel/api/clients/onlines", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"msg":"session expired"}`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	c, cookie := newTestClient(ts)
	if err := c.FetchOnlineUsersCount(cookie); err == nil {
		t.Fatal("FetchOnlineUsersCount: expected error on success:false response, got nil")
	}
}

func TestFetchInboundsList(t *testing.T) {
	// settings/streamSettings/sniffing are nested OBJECTS here (the v3 shape);
	// the decode must not choke on them the way an old string-typed Settings field would.
	const body = `{"success":true,"obj":[
		{"id":1,"up":100,"down":200,"remark":"VLESS-443",
		 "settings":{"clients":[{"email":"alice"}],"decryption":"none"},
		 "streamSettings":{"network":"tcp","security":"reality"},
		 "sniffing":{"enabled":true,"destOverride":["http","tls"]},
		 "clientStats":[{"id":5,"email":"alice","up":50,"down":60}]}
	]}`
	mux := http.NewServeMux()
	mux.HandleFunc("/panel/api/inbounds/list", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	c, cookie := newTestClient(ts)
	if err := c.FetchInboundsList(cookie); err != nil {
		t.Fatalf("FetchInboundsList: %v", err)
	}
	if got := testutil.ToFloat64(metrics.InboundUp.WithLabelValues("1", "VLESS-443")); got != 100 {
		t.Errorf("InboundUp = %v, want 100", got)
	}
	if got := testutil.ToFloat64(metrics.InboundDown.WithLabelValues("1", "VLESS-443")); got != 200 {
		t.Errorf("InboundDown = %v, want 200", got)
	}
	if got := testutil.ToFloat64(metrics.ClientUp.WithLabelValues("5", "alice")); got != 50 {
		t.Errorf("ClientUp = %v, want 50", got)
	}
	if got := testutil.ToFloat64(metrics.ClientDown.WithLabelValues("5", "alice")); got != 60 {
		t.Errorf("ClientDown = %v, want 60", got)
	}
}

func TestFetchServerStatus(t *testing.T) {
	const body = `{"success":true,"obj":{
		"cpu":12.5,"mem":{"current":100,"total":200},
		"xray":{"state":"running","version":"v25.10.31"},
		"appStats":{"threads":42,"mem":123456,"uptime":86400}}}`
	mux := http.NewServeMux()
	mux.HandleFunc("/panel/api/server/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte(body))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	c, cookie := newTestClient(ts)
	if err := c.FetchServerStatus(cookie); err != nil {
		t.Fatalf("FetchServerStatus: %v", err)
	}
	if got := testutil.ToFloat64(metrics.PanelThreads); got != 42 {
		t.Errorf("PanelThreads = %v, want 42", got)
	}
	if got := testutil.ToFloat64(metrics.PanelMemory); got != 123456 {
		t.Errorf("PanelMemory = %v, want 123456", got)
	}
	if got := testutil.ToFloat64(metrics.PanelUptime); got != 86400 {
		t.Errorf("PanelUptime = %v, want 86400", got)
	}
	// v-prefixed version is not numeric: the label carries the version, the
	// value is best-effort 0. This preserves the pre-v3 behavior exactly.
	if got := testutil.ToFloat64(metrics.XrayVersion.WithLabelValues("v25.10.31")); got != 0 {
		t.Errorf("XrayVersion value = %v, want 0", got)
	}
}

func resetAuthCache() {
	authCache.Lock()
	authCache.Cookie = http.Cookie{}
	authCache.CSRFToken = ""
	authCache.ExpiresAt = time.Time{}
	authCache.Unlock()
}

func TestCSRFHeaderOnGet(t *testing.T) {
	resetAuthCache()
	authCache.Lock()
	authCache.CSRFToken = "test-token"
	authCache.Unlock()
	t.Cleanup(resetAuthCache)

	var gotToken string
	mux := http.NewServeMux()
	mux.HandleFunc("/panel/api/server/status", func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-CSRF-Token")
		_, _ = w.Write([]byte(`{"success":true,"obj":{"xray":{"version":"1.0"},"appStats":{"threads":1,"mem":1,"uptime":1}}}`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	c, cookie := newTestClient(ts)
	if err := c.FetchServerStatus(cookie); err != nil {
		t.Fatalf("FetchServerStatus: %v", err)
	}
	if gotToken != "test-token" {
		t.Errorf("X-CSRF-Token on GET = %q, want %q", gotToken, "test-token")
	}
}

func TestGetAuthTokenRequiresCSRF(t *testing.T) {
	resetAuthCache()
	t.Cleanup(resetAuthCache)

	// No /csrf-token handler → the mint returns "" → login must be refused.
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		t.Error("/login must not be called when no CSRF token is available")
		http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "x"})
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok"}`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	c := NewAPIClient(APIConfig{BaseURL: ts.URL, ApiUsername: "u", ApiPassword: "p"})
	if _, err := c.GetAuthToken(); err == nil {
		t.Fatal("expected error when CSRF token is unavailable, got nil")
	}
}

func TestGetAuthTokenSuccess(t *testing.T) {
	resetAuthCache()
	t.Cleanup(resetAuthCache)

	mux := http.NewServeMux()
	mux.HandleFunc("/csrf-token", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "csrf-sess"})
		_, _ = w.Write([]byte(`{"success":true,"obj":"tok123"}`))
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-CSRF-Token"); got != "tok123" {
			t.Errorf("login X-CSRF-Token = %q, want tok123", got)
		}
		if c, err := r.Cookie("3x-ui"); err != nil || c.Value != "csrf-sess" {
			t.Errorf("login must carry the /csrf-token session cookie; got %v (err %v)", c, err)
		}
		http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "auth-sess"})
		_, _ = w.Write([]byte(`{"success":true,"msg":"ok"}`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	c := NewAPIClient(APIConfig{BaseURL: ts.URL, ApiUsername: "u", ApiPassword: "p"})
	cookie, err := c.GetAuthToken()
	if err != nil {
		t.Fatalf("GetAuthToken: %v", err)
	}
	if cookie.Value != "auth-sess" {
		t.Errorf("cookie = %q, want auth-sess", cookie.Value)
	}
	authCache.Lock()
	tok := authCache.CSRFToken
	authCache.Unlock()
	if tok != "tok123" {
		t.Errorf("cached CSRFToken = %q, want tok123", tok)
	}
}
