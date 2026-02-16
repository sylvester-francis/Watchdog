package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// --- login ---

func cmdLogin(args []string) {
	args = filterArgs(args)

	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println(`Usage: watchdog login

Saves hub URL and API token to ~/.watchdog/config.json.
Generate a token from the WatchDog web UI under Settings > API Tokens.`)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Hub URL (e.g. http://localhost:8080): ")
	hubURL, _ := reader.ReadString('\n')
	hubURL = strings.TrimSpace(hubURL)
	hubURL = strings.TrimRight(hubURL, "/")

	if hubURL == "" {
		fatal("hub URL is required")
	}

	fmt.Print("API Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		fatal("API token is required")
	}

	// Validate by making a test request
	cfg := &CLIConfig{HubURL: hubURL, Token: token}
	client := newClient(cfg)
	_, status, err := client.get("/dashboard/stats")
	if err != nil {
		fatal("could not connect to hub: %v", err)
	}
	if status == 401 {
		fatal("invalid API token")
	}
	if status != 200 {
		fatal("unexpected response from hub (HTTP %d)", status)
	}

	if err := saveConfig(cfg); err != nil {
		fatal("save config: %v", err)
	}

	fmt.Println("Logged in successfully.")
}

// --- monitors ---

func cmdMonitors(args []string) {
	args = filterArgs(args)

	if len(args) == 0 {
		monitorsHelp()
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list", "ls":
		monitorsList()
	case "get":
		if len(subArgs) < 1 {
			fatal("usage: watchdog monitors get <id>")
		}
		monitorsGet(subArgs[0])
	case "create":
		monitorsCreate(subArgs)
	case "delete", "rm":
		if len(subArgs) < 1 {
			fatal("usage: watchdog monitors delete <id>")
		}
		monitorsDelete(subArgs[0])
	case "--help", "-h":
		monitorsHelp()
	default:
		fatal("unknown monitors subcommand: %s", sub)
	}
}

func monitorsHelp() {
	fmt.Println(`Usage: watchdog monitors <subcommand>

Subcommands:
  list                     List all monitors
  get <id>                 Show monitor details
  create                   Create a monitor (interactive)
  delete <id>              Delete a monitor

Flags:
  --json                   Output as JSON`)
}

func monitorsList() {
	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.get("/monitors")
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal(apiError(body, status))
	}

	var resp struct {
		Data []struct {
			ID       string `json:"id"`
			AgentID  string `json:"agent_id"`
			Name     string `json:"name"`
			Type     string `json:"type"`
			Target   string `json:"target"`
			Status   string `json:"status"`
			Enabled  bool   `json:"enabled"`
			Interval int    `json:"interval_seconds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		fatal("parse response: %v", err)
	}

	if jsonOutput {
		printJSON(resp.Data)
		return
	}

	headers := []string{"ID", "NAME", "TYPE", "TARGET", "STATUS", "INTERVAL"}
	var rows [][]string
	for _, m := range resp.Data {
		status := m.Status
		if !m.Enabled {
			status = "disabled"
		}
		rows = append(rows, []string{
			m.ID[:8], m.Name, m.Type, truncate(m.Target, 40), status, fmt.Sprintf("%ds", m.Interval),
		})
	}

	if len(rows) == 0 {
		fmt.Println("No monitors found.")
		return
	}
	printTable(headers, rows)
}

func monitorsGet(id string) {
	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.get("/monitors/" + id)
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal(apiError(body, status))
	}

	if jsonOutput {
		var raw json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			fatal("parse response: %v", err)
		}
		printJSON(raw)
		return
	}

	var resp struct {
		Data struct {
			ID       string `json:"id"`
			AgentID  string `json:"agent_id"`
			Name     string `json:"name"`
			Type     string `json:"type"`
			Target   string `json:"target"`
			Status   string `json:"status"`
			Enabled  bool   `json:"enabled"`
			Interval int    `json:"interval_seconds"`
			Timeout  int    `json:"timeout_seconds"`
		} `json:"data"`
		Heartbeats struct {
			UptimeUp   int `json:"uptime_up"`
			UptimeDown int `json:"uptime_down"`
			Total      int `json:"total"`
		} `json:"heartbeats"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		fatal("parse response: %v", err)
	}

	d := resp.Data
	fmt.Printf("ID:        %s\n", d.ID)
	fmt.Printf("Name:      %s\n", d.Name)
	fmt.Printf("Type:      %s\n", d.Type)
	fmt.Printf("Target:    %s\n", d.Target)
	fmt.Printf("Status:    %s\n", d.Status)
	fmt.Printf("Enabled:   %v\n", d.Enabled)
	fmt.Printf("Interval:  %ds\n", d.Interval)
	fmt.Printf("Timeout:   %ds\n", d.Timeout)
	fmt.Printf("Agent:     %s\n", d.AgentID)

	hb := resp.Heartbeats
	if hb.Total > 0 {
		uptime := float64(hb.UptimeUp) / float64(hb.Total) * 100
		fmt.Printf("Uptime:    %.1f%% (%d/%d)\n", uptime, hb.UptimeUp, hb.Total)
	}
}

func monitorsCreate(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println(`Usage: watchdog monitors create

Creates a monitor interactively. You'll be prompted for:
  - agent_id: The agent UUID to assign the monitor to
  - name: A human-readable name
  - type: http, tcp, ping, or dns
  - target: The address to check`)
		return
	}

	cfg := mustLoadConfig()
	client := newClient(cfg)
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Agent ID: ")
	agentID, _ := reader.ReadString('\n')
	agentID = strings.TrimSpace(agentID)

	fmt.Print("Name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Type (http/tcp/ping/dns): ")
	monType, _ := reader.ReadString('\n')
	monType = strings.TrimSpace(monType)

	fmt.Print("Target: ")
	target, _ := reader.ReadString('\n')
	target = strings.TrimSpace(target)

	if agentID == "" || name == "" || monType == "" || target == "" {
		fatal("all fields are required")
	}

	reqBody := map[string]interface{}{
		"agent_id": agentID,
		"name":     name,
		"type":     monType,
		"target":   target,
	}

	body, status, err := client.post("/monitors", reqBody)
	if err != nil {
		fatal("%v", err)
	}
	if status != 201 {
		fatal(apiError(body, status))
	}

	var resp struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		fatal("parse response: %v", err)
	}

	fmt.Printf("Monitor created: %s (%s)\n", resp.Data.Name, resp.Data.ID)
}

func monitorsDelete(id string) {
	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.delete("/monitors/" + id)
	if err != nil {
		fatal("%v", err)
	}
	if status != 204 {
		fatal(apiError(body, status))
	}

	fmt.Println("Monitor deleted.")
}

// --- agents ---

func cmdAgents(args []string) {
	args = filterArgs(args)

	if len(args) == 0 {
		agentsHelp()
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list", "ls":
		agentsList()
	case "create":
		agentsCreate(subArgs)
	case "delete", "rm":
		if len(subArgs) < 1 {
			fatal("usage: watchdog agents delete <id>")
		}
		agentsDelete(subArgs[0])
	case "--help", "-h":
		agentsHelp()
	default:
		fatal("unknown agents subcommand: %s", sub)
	}
}

func agentsHelp() {
	fmt.Println(`Usage: watchdog agents <subcommand>

Subcommands:
  list                     List all agents
  create <name>            Create an agent (returns API key)
  delete <id>              Delete an agent

Flags:
  --json                   Output as JSON`)
}

func agentsList() {
	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.get("/agents")
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal(apiError(body, status))
	}

	var resp struct {
		Data []struct {
			ID         string  `json:"id"`
			Name       string  `json:"name"`
			Status     string  `json:"status"`
			LastSeenAt *string `json:"last_seen_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		fatal("parse response: %v", err)
	}

	if jsonOutput {
		printJSON(resp.Data)
		return
	}

	headers := []string{"ID", "NAME", "STATUS", "LAST SEEN"}
	var rows [][]string
	for _, a := range resp.Data {
		lastSeen := "never"
		if a.LastSeenAt != nil {
			lastSeen = *a.LastSeenAt
		}
		rows = append(rows, []string{a.ID[:8], a.Name, a.Status, lastSeen})
	}

	if len(rows) == 0 {
		fmt.Println("No agents found.")
		return
	}
	printTable(headers, rows)
}

func agentsCreate(args []string) {
	if len(args) < 1 {
		fatal("usage: watchdog agents create <name>")
	}

	name := args[0]
	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.post("/agents", map[string]string{"name": name})
	if err != nil {
		fatal("%v", err)
	}
	if status != 201 {
		fatal(apiError(body, status))
	}

	var resp struct {
		Data struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			APIKey string `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		fatal("parse response: %v", err)
	}

	if jsonOutput {
		printJSON(resp.Data)
		return
	}

	fmt.Printf("Agent created: %s (%s)\n", resp.Data.Name, resp.Data.ID)
	fmt.Printf("API Key: %s\n", resp.Data.APIKey)
	fmt.Println("\nSave this API key — it cannot be retrieved again.")
}

func agentsDelete(id string) {
	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.delete("/agents/" + id)
	if err != nil {
		fatal("%v", err)
	}
	if status != 204 {
		fatal(apiError(body, status))
	}

	fmt.Println("Agent deleted.")
}

// --- incidents ---

func cmdIncidents(args []string) {
	args = filterArgs(args)

	if len(args) == 0 {
		incidentsHelp()
		return
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list", "ls":
		incidentsList(subArgs)
	case "ack", "acknowledge":
		if len(subArgs) < 1 {
			fatal("usage: watchdog incidents ack <id>")
		}
		incidentsAck(subArgs[0])
	case "resolve":
		if len(subArgs) < 1 {
			fatal("usage: watchdog incidents resolve <id>")
		}
		incidentsResolve(subArgs[0])
	case "--help", "-h":
		incidentsHelp()
	default:
		fatal("unknown incidents subcommand: %s", sub)
	}
}

func incidentsHelp() {
	fmt.Println(`Usage: watchdog incidents <subcommand>

Subcommands:
  list [--resolved]        List incidents (default: active)
  ack <id>                 Acknowledge an incident
  resolve <id>             Resolve an incident

Flags:
  --json                   Output as JSON`)
}

func incidentsList(args []string) {
	cfg := mustLoadConfig()
	client := newClient(cfg)

	path := "/incidents"
	for _, a := range args {
		if a == "--resolved" {
			path = "/incidents?status=resolved"
			break
		}
	}

	body, status, err := client.get(path)
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal(apiError(body, status))
	}

	var resp struct {
		Data []struct {
			ID             string  `json:"id"`
			MonitorID      string  `json:"monitor_id"`
			Status         string  `json:"status"`
			StartedAt      string  `json:"started_at"`
			ResolvedAt     *string `json:"resolved_at"`
			AcknowledgedAt *string `json:"acknowledged_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		fatal("parse response: %v", err)
	}

	if jsonOutput {
		printJSON(resp.Data)
		return
	}

	headers := []string{"ID", "MONITOR", "STATUS", "STARTED"}
	var rows [][]string
	for _, i := range resp.Data {
		rows = append(rows, []string{i.ID[:8], i.MonitorID[:8], i.Status, i.StartedAt})
	}

	if len(rows) == 0 {
		fmt.Println("No incidents found.")
		return
	}
	printTable(headers, rows)
}

func incidentsAck(id string) {
	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.post("/incidents/"+id+"/acknowledge", nil)
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal(apiError(body, status))
	}

	fmt.Println("Incident acknowledged.")
}

func incidentsResolve(id string) {
	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.post("/incidents/"+id+"/resolve", nil)
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal(apiError(body, status))
	}

	fmt.Println("Incident resolved.")
}

// --- status ---

func cmdStatus(args []string) {
	args = filterArgs(args)

	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println(`Usage: watchdog status

Shows a summary of your infrastructure: agents, monitors, and active incidents.`)
		return
	}

	cfg := mustLoadConfig()
	client := newClient(cfg)

	body, status, err := client.get("/dashboard/stats")
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal(apiError(body, status))
	}

	var stats struct {
		TotalMonitors   int `json:"total_monitors"`
		MonitorsUp      int `json:"monitors_up"`
		MonitorsDown    int `json:"monitors_down"`
		ActiveIncidents int `json:"active_incidents"`
		TotalAgents     int `json:"total_agents"`
		OnlineAgents    int `json:"online_agents"`
	}
	if err := json.Unmarshal(body, &stats); err != nil {
		fatal("parse response: %v", err)
	}

	if jsonOutput {
		printJSON(stats)
		return
	}

	fmt.Printf("Agents:     %d online / %d total\n", stats.OnlineAgents, stats.TotalAgents)
	fmt.Printf("Monitors:   %d up / %d down / %d total\n", stats.MonitorsUp, stats.MonitorsDown, stats.TotalMonitors)
	fmt.Printf("Incidents:  %d active\n", stats.ActiveIncidents)
}

// --- helpers ---

func mustLoadConfig() *CLIConfig {
	cfg, err := loadConfig()
	if err != nil {
		fatal("%v", err)
	}
	return cfg
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
