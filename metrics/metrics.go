package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// User-related metrics
	OnlineUsersCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "x_ui_total_online_users",
			Help: "Total number of online users",
		},
	)
	// Inbound-related metrics
	InboundUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "x_ui_inbound_up_bytes",
			Help: "Total uploaded bytes per inbound",
		}, []string{"id", "remark"},
	)
	InboundDown = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "x_ui_inbound_down_bytes",
			Help: "Total downloaded bytes per inbound",
		}, []string{"id", "remark"},
	)
	// Client-related metrics
	ClientUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "x_ui_client_up_bytes",
			Help: "Total uploaded bytes per client",
		}, []string{"id", "email"},
	)
	ClientDown = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "x_ui_client_down_bytes",
			Help: "Total downloaded bytes per client",
		}, []string{"id", "email"},
	)
	// System-related metrics
	XrayVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "x_ui_xray_version",
			Help: "XRay version used by 3X-UI",
		},
		[]string{"version"},
	)
	PanelGoroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "x_ui_panel_goroutines",
			Help: "Goroutines running in the 3X-UI panel process (appStats.threads)",
		},
	)
	PanelMemoryBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "x_ui_panel_memory_bytes",
			Help: "Resident memory of the 3X-UI panel process in bytes (appStats.mem)",
		},
	)
	// Info-style: value is always 1, the version lives in the label.
	PanelVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "x_ui_panel_version",
			Help: "3X-UI panel version, always 1; read the version label",
		},
		[]string{"version"},
	)
	// The signal to alert on; only the panel's "running" state counts as up.
	XrayUp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "x_ui_xray_up",
			Help: "1 when the Xray process is running, 0 when stopped or errored",
		},
	)
	// Separate from XrayUp so free-form error text never lands on the alert metric.
	XrayState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "x_ui_xray_state",
			Help: "Xray process state reported by the panel, always 1; read the state and error labels",
		},
		[]string{"state", "error"},
	)
	AmneziaWGUp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "x_ui_amneziawg_up",
			Help: "1 when the embedded AmneziaWG interface is running, 0 otherwise",
		},
	)
	XrayUptimeSeconds = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "x_ui_xray_uptime_seconds",
			Help: "Uptime of the Xray process in seconds, 0 when Xray is stopped (appStats.uptime)",
		},
	)
)
