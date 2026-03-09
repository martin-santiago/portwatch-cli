package main

import (
	"fmt"
	"strings"
	"time"
)

type ViewMode int

const (
	ViewPorts ViewMode = iota
	ViewFilters
	ViewAddPort
)

type AppState struct {
	entries      []PortEntry
	allEntries   []PortEntry
	cursor       int
	mode         ViewMode
	filterCursor int
	addPortBuf   string
	message      string
	messageTime  time.Time
	lastRefresh  time.Time
}

func newAppState() *AppState {
	return &AppState{
		mode: ViewPorts,
	}
}

func (s *AppState) setMessage(msg string) {
	s.message = msg
	s.messageTime = time.Now()
}

func (s *AppState) refresh() {
	entries, err := ScanPorts()
	if err != nil {
		s.setMessage(fmt.Sprintf("Error: %v", err))
		return
	}

	s.allEntries = entries
	cfg := getConfig()
	if cfg.FilterEnabled {
		s.entries = FilterPorts(entries, cfg.FilterPorts)
	} else {
		s.entries = entries
	}

	if s.cursor >= len(s.entries) {
		s.cursor = len(s.entries) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}

	s.lastRefresh = time.Now()
}

func (s *AppState) render() {
	w, h := getTerminalSize()
	var buf strings.Builder

	buf.WriteString(clearScreen)
	buf.WriteString(fmt.Sprintf(moveTo, 1, 1))

	cfg := getConfig()

	switch s.mode {
	case ViewPorts:
		s.renderPortsView(&buf, w, h, cfg)
	case ViewFilters:
		s.renderFiltersView(&buf, w, h, cfg)
	case ViewAddPort:
		s.renderAddPortView(&buf, w, h)
	}

	fmt.Print(buf.String())
}

func (s *AppState) renderPortsView(buf *strings.Builder, w, h int, cfg Config) {
	// ── Header ──
	title := " PortWatch CLI "
	filterBadge := ""
	if cfg.FilterEnabled {
		filterBadge = fmt.Sprintf(" %s%s FILTER ON %s", bgGreen, bold, reset)
	}

	buf.WriteString(fmt.Sprintf("%s%s%s%s%s\r\n", bgBlue, bold, white, padRight(title, w), reset))

	// Status line
	totalPorts := len(s.allEntries)
	shownPorts := len(s.entries)
	statusLine := fmt.Sprintf(" %d ports listening", totalPorts)
	if cfg.FilterEnabled {
		statusLine = fmt.Sprintf(" %d/%d ports (filtered)", shownPorts, totalPorts)
	}
	buf.WriteString(fmt.Sprintf("%s%s%s%s\r\n", dim, statusLine, filterBadge, reset))
	buf.WriteString("\r\n")

	// ── Column headers ──
	colPort := 8
	colPID := 10
	colProcess := 20
	colUser := 15

	header := fmt.Sprintf(" %s  %s  %s  %s",
		padRight("PORT", colPort),
		padRight("PID", colPID),
		padRight("PROCESS", colProcess),
		padRight("USER", colUser),
	)
	buf.WriteString(fmt.Sprintf("%s%s%s%s\r\n", bold, cyan, padRight(header, w), reset))
	buf.WriteString(fmt.Sprintf("%s%s%s\r\n", dim, strings.Repeat("─", w), reset))

	// ── Port rows ──
	if len(s.entries) == 0 {
		buf.WriteString("\r\n")
		if cfg.FilterEnabled {
			buf.WriteString(fmt.Sprintf("  %s%sNo ports matching your filter.%s\r\n", dim, yellow, reset))
			buf.WriteString(fmt.Sprintf("  %sPress %sf%s%s to toggle filter off, or %se%s%s to edit filters.%s\r\n",
				dim, bold, reset, dim, bold, reset, dim, reset))
		} else {
			buf.WriteString(fmt.Sprintf("  %s%sNo listening ports found.%s\r\n", dim, yellow, reset))
		}
	} else {
		// Calculate visible rows
		headerLines := 5
		footerLines := 5
		maxRows := h - headerLines - footerLines
		if maxRows < 1 {
			maxRows = 1
		}

		// Scroll offset
		startIdx := 0
		if s.cursor >= maxRows {
			startIdx = s.cursor - maxRows + 1
		}
		endIdx := startIdx + maxRows
		if endIdx > len(s.entries) {
			endIdx = len(s.entries)
		}

		for i := startIdx; i < endIdx; i++ {
			e := s.entries[i]

			portStr := fmt.Sprintf(":%d", e.Port)
			row := fmt.Sprintf(" %s  %s  %s  %s",
				padRight(portStr, colPort),
				padLeft(fmt.Sprintf("%d", e.PID), colPID),
				padRight(e.Process, colProcess),
				padRight(e.User, colUser),
			)

			if i == s.cursor {
				buf.WriteString(fmt.Sprintf("%s%s%s%s\r\n", inverse, bold, padRight(row, w), reset))
			} else {
				buf.WriteString(fmt.Sprintf("%s\r\n", padRight(row, w)))
			}
		}

		// Scroll indicator
		if len(s.entries) > maxRows {
			scrollInfo := fmt.Sprintf(" [%d/%d]", s.cursor+1, len(s.entries))
			buf.WriteString(fmt.Sprintf("%s%s%s\r\n", dim, scrollInfo, reset))
		}
	}

	// ── Message bar ──
	if s.message != "" && time.Since(s.messageTime) < 3*time.Second {
		buf.WriteString("\r\n")
		buf.WriteString(fmt.Sprintf(" %s%s%s%s\r\n", bold, yellow, s.message, reset))
	}

	// ── Footer ──
	buf.WriteString(fmt.Sprintf("\r\n%s", strings.Repeat("─", w)))
	buf.WriteString("\r\n")

	keys := []struct {
		key  string
		desc string
	}{
		{"↑/↓/j/k", "navigate"},
		{"enter/x", "kill"},
		{"f", "filter"},
		{"e", "edit filters"},
		{"r", "refresh"},
		{"q", "quit"},
	}

	var keyParts []string
	for _, k := range keys {
		keyParts = append(keyParts, fmt.Sprintf("%s%s%s %s%s%s", bold, green, k.key, reset, dim, k.desc))
	}
	footer := " " + strings.Join(keyParts, fmt.Sprintf("%s  ", reset))
	buf.WriteString(fmt.Sprintf("%s%s\r\n", footer, reset))
}

func (s *AppState) renderFiltersView(buf *strings.Builder, w, _ int, cfg Config) {
	// Header
	buf.WriteString(fmt.Sprintf("%s%s%s%s%s\r\n", bgYellow, bold, white, padRight(" Edit Filters ", w), reset))
	buf.WriteString("\r\n")

	if len(cfg.FilterPorts) == 0 {
		buf.WriteString(fmt.Sprintf("  %s%sNo filter ports configured.%s\r\n", dim, yellow, reset))
		buf.WriteString(fmt.Sprintf("  %sPress %sa%s%s to add a port.%s\r\n", dim, bold, reset, dim, reset))
	} else {
		buf.WriteString(fmt.Sprintf("  %s%sWatched ports:%s\r\n\r\n", bold, cyan, reset))

		for i, port := range cfg.FilterPorts {
			label := fmt.Sprintf("  Port %d", port)
			if i == s.filterCursor {
				buf.WriteString(fmt.Sprintf("  %s%s %s %s\r\n", inverse, bold, padRight(label, 30), reset))
			} else {
				buf.WriteString(fmt.Sprintf("  %s\r\n", label))
			}
		}
	}

	// Message
	if s.message != "" && time.Since(s.messageTime) < 3*time.Second {
		buf.WriteString(fmt.Sprintf("\r\n %s%s%s%s\r\n", bold, yellow, s.message, reset))
	}

	// Footer
	buf.WriteString(fmt.Sprintf("\r\n%s\r\n", strings.Repeat("─", w)))

	keys := []struct {
		key  string
		desc string
	}{
		{"↑/↓", "navigate"},
		{"a", "add port"},
		{"d/x", "remove port"},
		{"esc", "back"},
	}

	var keyParts []string
	for _, k := range keys {
		keyParts = append(keyParts, fmt.Sprintf("%s%s%s %s%s%s", bold, green, k.key, reset, dim, k.desc))
	}
	footer := " " + strings.Join(keyParts, fmt.Sprintf("%s  ", reset))
	buf.WriteString(fmt.Sprintf("%s%s\r\n", footer, reset))
}

func (s *AppState) renderAddPortView(buf *strings.Builder, w, _ int) {
	buf.WriteString(fmt.Sprintf("%s%s%s%s%s\r\n", bgGreen, bold, white, padRight(" Add Port ", w), reset))
	buf.WriteString("\r\n")
	buf.WriteString(fmt.Sprintf("  Enter a port number (1-65535):\r\n\r\n"))
	buf.WriteString(fmt.Sprintf("  > %s%s%s%s%s\r\n", bold, white, s.addPortBuf, showCursor, reset))

	if s.message != "" && time.Since(s.messageTime) < 3*time.Second {
		buf.WriteString(fmt.Sprintf("\r\n  %s%s%s%s\r\n", bold, red, s.message, reset))
	}

	buf.WriteString(fmt.Sprintf("\r\n%s\r\n", strings.Repeat("─", w)))
	buf.WriteString(fmt.Sprintf(" %s%senter%s %sconfirm%s  %s%sesc%s %scancel%s\r\n",
		bold, green, reset, dim, reset, bold, green, reset, dim, reset))
}
