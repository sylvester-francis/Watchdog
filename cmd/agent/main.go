package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Flags
	hubURL := flag.String("hub", "ws://localhost:8080/ws/agent", "Hub WebSocket URL")
	apiKey := flag.String("api-key", "", "Agent API key")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("WatchDog Agent %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// API key from flag or environment
	key := *apiKey
	if key == "" {
		key = os.Getenv("WATCHDOG_API_KEY")
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: API key required. Use -api-key flag or WATCHDOG_API_KEY env var")
		os.Exit(1)
	}

	fmt.Printf("WatchDog Agent %s\n", Version)
	fmt.Printf("Connecting to hub: %s\n", *hubURL)

	// TODO: Implement WebSocket connection and task execution
	// For now, just wait for signal
	fmt.Println("Agent running. Press Ctrl+C to stop.")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nAgent stopped.")
}
