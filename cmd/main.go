package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"sol-speedtest/pkg/config"
	"sol-speedtest/pkg/logger"
	"sol-speedtest/pkg/utils"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to the configuration file")
	testInterval := flag.Int("interval", 3000, "Interval between test rounds in milliseconds")
	logLevel := flag.String("loglevel", "", "Log level: debug, info, warn, error")
	flag.Parse()

	cfg, err := config.LoadConfigFromFile(*configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	var level logger.LogLevel
	switch cfg.LogLevel {
	case "debug":
		level = logger.DebugLevel
	case "info":
		level = logger.InfoLevel
	case "warn":
		level = logger.WarnLevel
	case "error":
		level = logger.ErrorLevel
	default:
		level = logger.InfoLevel
	}

	privKey := os.Getenv("SIGNER")
	runner := utils.NewTestRunner(cfg, privKey, level)

	if *testInterval > 0 {
		runner.SetTestInterval(time.Duration(*testInterval) * time.Millisecond)
	}

	totalEndpoints := 0
	for _, provider := range cfg.Providers {
		totalEndpoints += len(provider.Endpoints)
	}

	fmt.Printf("Starting competition test with %d providers, %d total endpoints, and %d test rounds\n",
		len(cfg.Providers), totalEndpoints, cfg.TestCount)

	startTime := time.Now()

	if err := runner.RunTests(); err != nil {
		fmt.Printf("Error running tests: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	fmt.Printf("Tests completed in %v\n", duration)
	fmt.Printf("Results saved to directory: %s\n", cfg.OutputPath)
}
