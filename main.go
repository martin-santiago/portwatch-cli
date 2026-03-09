package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Handle non-interactive CLI flags
	if len(os.Args) > 1 {
		handleCLIArgs()
		return
	}

	// Interactive TUI mode
	runInteractive()
}

func handleCLIArgs() {
	initConfig()

	args := os.Args[1:]
	cmd := args[0]

	switch cmd {
	case "list", "ls", "l":
		listPorts(args[1:])
	case "kill", "k":
		killByPort(args[1:])
	case "filter", "f":
		manageFilter(args[1:])
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		fmt.Println("portwatch v1.0.0")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func listPorts(args []string) {
	entries, err := ScanPorts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning ports: %v\n", err)
		os.Exit(1)
	}

	cfg := getConfig()

	// Check for --filter flag
	useFilter := false
	for _, a := range args {
		if a == "--filter" || a == "-f" {
			useFilter = true
		}
	}

	if useFilter || cfg.FilterEnabled {
		entries = FilterPorts(entries, cfg.FilterPorts)
	}

	if len(entries) == 0 {
		fmt.Println("No listening ports found.")
		return
	}

	// Table output
	fmt.Printf("%-8s  %8s  %-20s  %-15s\n", "PORT", "PID", "PROCESS", "USER")
	fmt.Println(strings.Repeat("─", 56))

	for _, e := range entries {
		fmt.Printf(":%-7d  %8d  %-20s  %-15s\n", e.Port, e.PID, e.Process, e.User)
	}

	fmt.Printf("\n%d port(s) listening.\n", len(entries))
}

func killByPort(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: portwatch-cli kill <port|pid> [--pid]")
		os.Exit(1)
	}

	target, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid number: %s\n", args[0])
		os.Exit(1)
	}

	byPID := false
	for _, a := range args[1:] {
		if a == "--pid" {
			byPID = true
		}
	}

	if byPID {
		if err := KillProcess(target); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to kill PID %d: %v\n", target, err)
			os.Exit(1)
		}
		fmt.Printf("Killed PID %d\n", target)
		return
	}

	// Kill by port — find all processes on that port
	entries, err := ScanPorts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning ports: %v\n", err)
		os.Exit(1)
	}

	killed := 0
	for _, e := range entries {
		if e.Port == target {
			if err := KillProcess(e.PID); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to kill %s (PID %d) on port %d: %v\n",
					e.Process, e.PID, e.Port, err)
			} else {
				fmt.Printf("Killed %s (PID %d) on port %d\n", e.Process, e.PID, e.Port)
				killed++
			}
		}
	}

	if killed == 0 {
		fmt.Fprintf(os.Stderr, "No process found on port %d\n", target)
		os.Exit(1)
	}
}

func manageFilter(args []string) {
	if len(args) == 0 {
		// Show current filter config
		cfg := getConfig()
		status := "OFF"
		if cfg.FilterEnabled {
			status = "ON"
		}
		fmt.Printf("Filter mode: %s\n", status)
		if len(cfg.FilterPorts) > 0 {
			portStrs := make([]string, len(cfg.FilterPorts))
			for i, p := range cfg.FilterPorts {
				portStrs[i] = strconv.Itoa(p)
			}
			fmt.Printf("Watched ports: %s\n", strings.Join(portStrs, ", "))
		} else {
			fmt.Println("No watched ports configured.")
		}
		return
	}

	subcmd := args[0]
	switch subcmd {
	case "on":
		setFilterEnabled(true)
		fmt.Println("Filter mode enabled.")
	case "off":
		setFilterEnabled(false)
		fmt.Println("Filter mode disabled.")
	case "toggle":
		enabled := toggleFilter()
		if enabled {
			fmt.Println("Filter mode enabled.")
		} else {
			fmt.Println("Filter mode disabled.")
		}
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: portwatch-cli filter add <port>")
			os.Exit(1)
		}
		port, err := strconv.Atoi(args[1])
		if err != nil || port < 1 || port > 65535 {
			fmt.Fprintf(os.Stderr, "Invalid port: %s\n", args[1])
			os.Exit(1)
		}
		addFilterPort(port)
		fmt.Printf("Added port %d to filter.\n", port)
	case "remove", "rm":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: portwatch-cli filter remove <port>")
			os.Exit(1)
		}
		port, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid port: %s\n", args[1])
			os.Exit(1)
		}
		removeFilterPort(port)
		fmt.Printf("Removed port %d from filter.\n", port)
	default:
		fmt.Fprintf(os.Stderr, "Unknown filter command: %s\n", subcmd)
		fmt.Fprintln(os.Stderr, "Available: on, off, toggle, add <port>, remove <port>")
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`portwatch — Monitor listening ports and kill processes

USAGE
  portwatch                   Interactive TUI mode
  portwatch <command>         Non-interactive mode

COMMANDS
  list, ls            List all listening ports
  list --filter       List only filtered ports
  kill <port>         Kill all processes on a port
  kill <pid> --pid    Kill a specific process by PID
  filter              Show current filter config
  filter on|off       Enable/disable filter mode
  filter toggle       Toggle filter mode
  filter add <port>   Add a port to the watch list
  filter rm <port>    Remove a port from the watch list
  help                Show this help
  version             Show version

INTERACTIVE KEYS
  ↑/↓  j/k     Navigate port list
  enter/x      Kill selected process
  f            Toggle filter mode
  e            Edit filter list
  r            Force refresh
  q            Quit

CONFIG
  Filters are saved to ~/.portwatch.json

`)
}

// ─── Interactive TUI ────────────────────────────────────────────────────────

func runInteractive() {
	initConfig()
	enableRawMode()
	defer disableRawMode()

	state := newAppState()
	state.refresh()
	state.render()

	// Refresh ticker
	ticker := time.NewTicker(time.Duration(getConfig().RefreshIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Key input channel
	keyCh := make(chan byte, 1)
	go func() {
		for {
			k := readKey()
			if k != 0 {
				keyCh <- k
			}
		}
	}()

	for {
		select {
		case key := <-keyCh:
			if handleKey(state, key) {
				return // quit
			}
			state.render()

		case <-ticker.C:
			if state.mode == ViewPorts {
				state.refresh()
				state.render()
			}
		}
	}
}

func handleKey(s *AppState, key byte) bool {
	switch s.mode {
	case ViewPorts:
		return handlePortsKey(s, key)
	case ViewFilters:
		return handleFiltersKey(s, key)
	case ViewAddPort:
		return handleAddPortKey(s, key)
	}
	return false
}

func handlePortsKey(s *AppState, key byte) bool {
	switch key {
	case 'q', 3: // q or Ctrl+C
		return true

	case 'j': // Down (vim-style)
		if s.cursor < len(s.entries)-1 {
			s.cursor++
		}

	case 'k': // Up (vim-style)
		if s.cursor > 0 {
			s.cursor--
		}

	case 'x', 'X', 13: // Kill selected (x or Enter)
		killSelected(s)

	case 'f', 'F': // Toggle filter
		toggleFilter()
		s.refresh()
		s.setMessage("Filter " + map[bool]string{true: "ON", false: "OFF"}[getConfig().FilterEnabled])

	case 'e', 'E': // Edit filters
		s.mode = ViewFilters
		s.filterCursor = 0

	case 'r', 'R': // Refresh
		s.refresh()
		s.setMessage("Refreshed")
	}

	return false
}

func handleFiltersKey(s *AppState, key byte) bool {
	cfg := getConfig()

	switch key {
	case 27: // Esc — back to ports
		s.mode = ViewPorts
		fmt.Print(hideCursor)
		s.refresh()

	case 'q', 3:
		return true

	case 'j', 'J': // Down
		if s.filterCursor < len(cfg.FilterPorts)-1 {
			s.filterCursor++
		}

	case 'k': // Up
		if s.filterCursor > 0 {
			s.filterCursor--
		}

	case 'a', 'A': // Add port
		s.mode = ViewAddPort
		s.addPortBuf = ""
		fmt.Print(showCursor)

	case 'd', 'x', 127: // Delete/remove
		if len(cfg.FilterPorts) > 0 && s.filterCursor < len(cfg.FilterPorts) {
			port := cfg.FilterPorts[s.filterCursor]
			removeFilterPort(port)
			s.setMessage(fmt.Sprintf("Removed port %d", port))
			if s.filterCursor > 0 {
				s.filterCursor--
			}
		}
	}

	return false
}

func handleAddPortKey(s *AppState, key byte) bool {
	switch key {
	case 27: // Esc — back to filters
		s.mode = ViewFilters
		fmt.Print(hideCursor)

	case 13: // Enter — confirm
		port, err := strconv.Atoi(s.addPortBuf)
		if err != nil || port < 1 || port > 65535 {
			s.setMessage("Invalid port number (1-65535)")
			return false
		}
		addFilterPort(port)
		s.setMessage(fmt.Sprintf("Added port %d", port))
		s.mode = ViewFilters
		fmt.Print(hideCursor)

	case 127, 8: // Backspace
		if len(s.addPortBuf) > 0 {
			s.addPortBuf = s.addPortBuf[:len(s.addPortBuf)-1]
		}

	default:
		// Only accept digits
		if key >= '0' && key <= '9' && len(s.addPortBuf) < 5 {
			s.addPortBuf += string(key)
		}
	}

	return false
}

func killSelected(s *AppState) {
	if len(s.entries) == 0 || s.cursor >= len(s.entries) {
		return
	}

	entry := s.entries[s.cursor]
	err := KillProcess(entry.PID)
	if err != nil {
		s.setMessage(fmt.Sprintf("Failed to kill %s (PID %d): %v", entry.Process, entry.PID, err))
	} else {
		s.setMessage(fmt.Sprintf("Killed %s (PID %d) on port %d", entry.Process, entry.PID, entry.Port))
	}

	time.Sleep(300 * time.Millisecond)
	s.refresh()
}
