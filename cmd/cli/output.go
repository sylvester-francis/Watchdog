package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

var jsonOutput bool

func init() {
	for _, arg := range os.Args {
		if arg == "--json" {
			jsonOutput = true
			break
		}
	}
}

// filterArgs returns args with --json removed.
func filterArgs(args []string) []string {
	var filtered []string
	for _, a := range args {
		if a != "--json" {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// printJSON outputs data as formatted JSON.
func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// printTable prints a formatted table with headers and rows.
func printTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)

	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, col)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

// fatal prints an error and exits.
func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// apiError extracts an error message from API JSON response.
func apiError(body []byte, status int) string {
	var resp map[string]string
	if json.Unmarshal(body, &resp) == nil {
		if msg, ok := resp["error"]; ok {
			return msg
		}
	}
	return fmt.Sprintf("HTTP %d", status)
}
