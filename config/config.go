package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
)

type VersionFlag string

type CLI struct {
	Ip                 string      `name:"metrics-ip" help:"Ip to listen on" default:"0.0.0.0" env:"METRICS_IP"`
	Port               string      `name:"metrics-port" help:"Port to listen on" default:"9090" env:"METRICS_PORT"`
	ProtectedMetrics   bool        `name:"metrics-protected" help:"Whether metrics are protected by basic auth" default:"false" env:"METRICS_PROTECTED"`
	MetricsUsername    string      `name:"metrics-username" help:"Username for metrics if protected by basic auth" default:"metricsUser" env:"METRICS_USERNAME"`
	MetricsPassword    string      `name:"metrics-password" help:"Password for metrics if protected by basic auth" default:"MetricsVeryHardPassword" env:"METRICS_PASSWORD"`
	UpdateInterval     int         `name:"update-interval" help:"Interval for metrics update in seconds" default:"30" env:"UPDATE_INTERVAL"`
	ClientsBytesRows   int         `name:"clients-bytes-rows" help:"Limit rows for clients up/down bytes (0 = all data, else top N rows)" default:"0" env:"CLIENTS_BYTES_ROWS"`
	TimeZone           string      `name:"timezone" help:"Timezone used in the application" default:"UTC" env:"TIMEZONE"`
	BaseURL            string      `name:"panel-base-url" help:"Panel base URL" env:"PANEL_BASE_URL"`
	ApiUsername        string      `name:"panel-username" help:"Panel username" env:"PANEL_USERNAME"`
	ApiPassword        string      `name:"panel-password" help:"Panel password" env:"PANEL_PASSWORD"`
	InsecureSkipVerify bool        `name:"insecure-skip-verify" help:"Skip SSL certificate verification (INSECURE)" default:"false" env:"INSECURE_SKIP_VERIFY"`
	Version            VersionFlag `name:"version" help:"Print version information and quit"`
}

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Println("3X-UI Exporter (Fork)")
	fmt.Printf("Version:\t %s\n", vars["version"])
	fmt.Printf("Commit:\t %s\n", vars["commit"])
	fmt.Printf("Github (Marzban): https://github.com/kutovoys/marzban-exporter\n")
	fmt.Printf("GitHub (3X-UI Fork): https://github.com/hteppl/3x-ui-exporter\n")
	app.Exit(0)
	return nil
}

func Parse(version, commit string) (*CLI, error) {
	// Must run before parsing so kong can resolve the env tags.
	if err := loadEnvFile(); err != nil {
		return nil, fmt.Errorf("load env file: %w", err)
	}

	var config CLI
	_ = kong.Parse(&config,
		kong.Name("x-ui-exporter"),
		kong.Description("A command-line application for exporting 3X-UI metrics."),
		kong.Vars{
			"version": version,
			"commit":  commit,
		},
	)

	validatedConfig, err := validate(&config)
	if err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return validatedConfig, nil
}

func removeTrailingSlash(s string) string {
	if strings.HasSuffix(s, "/") {
		return strings.TrimSuffix(s, "/")
	}
	return s
}

func validate(config *CLI) (*CLI, error) {
	if config.BaseURL == "" {
		return nil, errors.New("x-ui-exporter: error: --panel-base-url must be provided")
	}
	if config.ApiUsername == "" {
		return nil, errors.New("x-ui-exporter: error: --panel-username must be provided")
	}
	if config.ApiPassword == "" {
		return nil, errors.New("x-ui-exporter: error: --panel-password must be provided")
	}
	config.BaseURL = removeTrailingSlash(config.BaseURL)
	return config, nil
}
