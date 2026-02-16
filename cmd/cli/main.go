package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "login":
		cmdLogin(args)
	case "monitors":
		cmdMonitors(args)
	case "agents":
		cmdAgents(args)
	case "incidents":
		cmdIncidents(args)
	case "status":
		cmdStatus(args)
	case "version":
		fmt.Printf("watchdog-cli v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: watchdog <command> [arguments]

Commands:
  login                    Authenticate with a WatchDog hub
  monitors                 Manage monitors (list, get, create, delete)
  agents                   Manage agents (list, create, delete)
  incidents                Manage incidents (list, ack, resolve)
  status                   Show infrastructure overview
  version                  Print version

Run 'watchdog <command> --help' for details on a command.`)
}
