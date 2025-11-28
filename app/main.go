package main

import (
	"local-path-exporter/collector"
	"local-path-exporter/parser"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration from environment variables
	storagePath := mustGetEnv("STORAGE_PATH")
	metricTemplate := mustGetEnv("METRIC_TEMPLATE")
	listenAddr := mustGetEnv("LISTEN_ADDR")
	intervalStr := mustGetEnv("REFRESH_INTERVAL_SECONDS")
	intervalSeconds, err := strconv.Atoi(intervalStr)
	if err != nil {
		log.Fatalf("CRITICAL: REFRESH_INTERVAL_SECONDS must be a valid integer: %v", err)
	}
	if intervalSeconds < 1 {
		log.Fatalf("CRITICAL: REFRESH_INTERVAL_SECONDS must be greater than 0")
	}

	refreshInterval := time.Duration(intervalSeconds) * time.Second

	log.Println("--- local-path Exporter Starting ---")
	log.Printf("Path: %s", storagePath)
	log.Printf("Template: %s", metricTemplate)
	log.Printf("Refresh Interval: %s", refreshInterval)

	// Validate storage path exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		log.Fatalf("CRITICAL: Storage path does not exist: %s", storagePath)
	}

	// Initialize parser and collector
	p, err := parser.NewDirParser(metricTemplate)
	if err != nil {
		log.Fatalf("CRITICAL: Invalid metric template: %v", err)
	}

	col := collector.NewPVCCollector(storagePath, p)

	// Register collector and start background scanner
	prometheus.MustRegister(col)
	col.StartBackgroundScanner(refreshInterval)

	// Start HTTP server
	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// mustGetEnv retrieves an environment variable or exits the application
func mustGetEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		log.Fatalf("CRITICAL: Missing required environment variable: %s", key)
	}
	return value
}
