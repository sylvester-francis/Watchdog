package main

import (
	"fmt"
	"os"
)

const version = "1.0.0"

// printDeprecationBanner is shown on every CLI invocation. Goes to
// stderr so JSON-mode output stays parseable. The CLI is scheduled
// for removal in v1.2.0; everything it does is doable via the REST
// API at /api/v1 with a bearer token, see the README's "Scripting
// with the API" section.
func printDeprecationBanner() {
	fmt.Fprintln(os.Stderr, "DEPRECATED: watchdog-cli is deprecated and will be removed in v1.2.0.")
	fmt.Fprintln(os.Stderr, "             Use the REST API directly. See:")
	fmt.Fprintln(os.Stderr, "             https://github.com/sylvester-francis/Watchdog#scripting-with-the-api")
	fmt.Fprintln(os.Stderr, "")
}

func main() {
	printDeprecationBanner()

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
