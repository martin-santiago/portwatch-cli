package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

type PortEntry struct {
	Port    int
	PID     int
	Process string
	User    string
}

func ScanPorts() ([]PortEntry, error) {
	if runtime.GOOS == "darwin" {
		return scanPortsDarwin()
	}
	return scanPortsLinux()
}

func scanPortsDarwin() ([]PortEntry, error) {
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-nP")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("lsof failed: %w", err)
	}
	return parseLsofOutput(string(out)), nil
}

func scanPortsLinux() ([]PortEntry, error) {
	// Try ss first (modern), fallback to netstat
	cmd := exec.Command("ss", "-tlnp")
	out, err := cmd.Output()
	if err == nil {
		return parseSsOutput(string(out)), nil
	}

	cmd = exec.Command("netstat", "-tlnp")
	out, err = cmd.Output()
	if err != nil {
		// Fallback to lsof
		return scanPortsDarwin()
	}
	return parseNetstatOutput(string(out)), nil
}

func parseLsofOutput(output string) []PortEntry {
	seen := make(map[string]bool)
	var entries []PortEntry

	lines := strings.Split(output, "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		command := fields[0]
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		user := fields[2]
		name := fields[len(fields)-1]

		port := extractPort(name)
		if port == 0 {
			continue
		}

		key := fmt.Sprintf("%d:%d", port, pid)
		if seen[key] {
			continue
		}
		seen[key] = true

		entries = append(entries, PortEntry{
			Port:    port,
			PID:     pid,
			Process: command,
			User:    user,
		})
	}

	sortEntries(entries)
	return entries
}

func parseSsOutput(output string) []PortEntry {
	seen := make(map[string]bool)
	var entries []PortEntry

	lines := strings.Split(output, "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		localAddr := fields[3]
		port := extractPort(localAddr)
		if port == 0 {
			continue
		}

		pid := 0
		process := "unknown"
		// Parse users:(("name",pid=XXX,fd=N)) format
		if len(fields) >= 6 {
			pidProcess := fields[5]
			pid, process = parseSsPidField(pidProcess)
		}

		key := fmt.Sprintf("%d:%d", port, pid)
		if seen[key] {
			continue
		}
		seen[key] = true

		entries = append(entries, PortEntry{
			Port:    port,
			PID:     pid,
			Process: process,
			User:    "-",
		})
	}

	sortEntries(entries)
	return entries
}

func parseSsPidField(field string) (int, string) {
	// Format: users:(("node",pid=12345,fd=20))
	process := "unknown"
	pid := 0

	if idx := strings.Index(field, "((\""); idx >= 0 {
		rest := field[idx+3:]
		if end := strings.Index(rest, "\""); end >= 0 {
			process = rest[:end]
		}
	}

	if idx := strings.Index(field, "pid="); idx >= 0 {
		rest := field[idx+4:]
		if end := strings.IndexAny(rest, ",)"); end >= 0 {
			pid, _ = strconv.Atoi(rest[:end])
		}
	}

	return pid, process
}

func parseNetstatOutput(output string) []PortEntry {
	seen := make(map[string]bool)
	var entries []PortEntry

	lines := strings.Split(output, "\n")
	for _, line := range lines[2:] {
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		if !strings.Contains(fields[5], "LISTEN") {
			continue
		}

		localAddr := fields[3]
		port := extractPort(localAddr)
		if port == 0 {
			continue
		}

		pid := 0
		process := "unknown"
		pidProgram := fields[6]
		if parts := strings.SplitN(pidProgram, "/", 2); len(parts) == 2 {
			pid, _ = strconv.Atoi(parts[0])
			process = parts[1]
		}

		key := fmt.Sprintf("%d:%d", port, pid)
		if seen[key] {
			continue
		}
		seen[key] = true

		entries = append(entries, PortEntry{
			Port:    port,
			PID:     pid,
			Process: process,
			User:    "-",
		})
	}

	sortEntries(entries)
	return entries
}

func extractPort(addr string) int {
	if idx := strings.LastIndex(addr, "]:"); idx >= 0 {
		p, _ := strconv.Atoi(addr[idx+2:])
		return p
	}
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		p, _ := strconv.Atoi(addr[idx+1:])
		return p
	}
	return 0
}

func sortEntries(entries []PortEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Port != entries[j].Port {
			return entries[i].Port < entries[j].Port
		}
		return entries[i].PID < entries[j].PID
	})
}

func KillProcess(pid int) error {
	err := syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		return syscall.Kill(pid, syscall.SIGKILL)
	}
	return nil
}

func FilterPorts(entries []PortEntry, ports []int) []PortEntry {
	if len(ports) == 0 {
		return entries
	}
	portSet := make(map[int]bool, len(ports))
	for _, p := range ports {
		portSet[p] = true
	}
	var filtered []PortEntry
	for _, e := range entries {
		if portSet[e.Port] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
