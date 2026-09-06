package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"x-ui-exporter/api"
	"x-ui-exporter/config"
	"x-ui-exporter/metrics"

	"github.com/go-co-op/gocron/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version = "unknown"
	commit  = "unknown"
)

func init() { //
	prometheus.MustRegister(
		// User-related metrics
		metrics.OnlineUsersCount,
		// Client-related metrics
		metrics.InboundUp,
		metrics.InboundDown,
		metrics.ClientUp,
		metrics.ClientDown,
		// System-related metrics
		metrics.XrayVersion,
		metrics.PanelThreads,
		metrics.PanelMemory,
		metrics.PanelUptime,
	)
}

func BasicAuthMiddleware(username, password string, protectedMetrics bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if protectedMetrics {
				user, pass, ok := r.BasicAuth()
				if !ok || user != username || pass != password {
					w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
					http.Error(w, "Unauthorized.", http.StatusUnauthorized)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	cliConfig, err := config.Parse(version, commit)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("3X-UI Exporter (https://github.com/hteppl/3x-ui-exporter/)", version)

	s, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		log.Fatalf("Create scheduler: %v", err)
	}
	defer func() {
		if err := s.Shutdown(); err != nil {
			log.Printf("Shutdown scheduler: %v", err)
		}
	}()

	client := api.NewAPIClient(api.APIConfig{
		BaseURL:            cliConfig.BaseURL,
		ApiUsername:        cliConfig.ApiUsername,
		ApiPassword:        cliConfig.ApiPassword,
		InsecureSkipVerify: cliConfig.InsecureSkipVerify,
		ClientsBytesRows:   cliConfig.ClientsBytesRows,
	})

	_, err = s.NewJob(
		gocron.DurationJob(time.Duration(cliConfig.UpdateInterval)*time.Second),
		gocron.NewTask(func() {
			token, err := client.GetAuthToken()
			if err != nil {
				log.Printf("get auth token: %v", err)
				return
			}

			// non-blocking errors
			if err := client.FetchOnlineUsersCount(token); err != nil {
				log.Printf("Error FetchOnlineUsersCount: %v", err)
			}

			if err := client.FetchServerStatus(token); err != nil {
				log.Printf("Error FetchServerStatus: %v", err)
			}

			if err := client.FetchInboundsList(token); err != nil {
				log.Printf("Error FetchInboundsList: %v", err)
			}
		}),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		log.Fatalf("Schedule job: %v", err)
	}

	s.Start()

	http.Handle("/metrics", BasicAuthMiddleware(
		cliConfig.MetricsUsername,
		cliConfig.MetricsPassword,
		cliConfig.ProtectedMetrics,
	)(promhttp.Handler()))

	log.Printf("Listening %s:%s", cliConfig.Ip, cliConfig.Port)
	log.Fatal(http.ListenAndServe(cliConfig.Ip+":"+cliConfig.Port, nil))
}
