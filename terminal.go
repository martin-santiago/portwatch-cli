package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// ANSI escape codes
const (
	clearScreen = "\033[2J"
	moveTo      = "\033[%d;%dH"
	bold        = "\033[1m"
	dim         = "\033[2m"
	reset       = "\033[0m"
	red         = "\033[31m"
	green       = "\033[32m"
	yellow      = "\033[33m"
	blue        = "\033[34m"
	magenta     = "\033[35m"
	cyan        = "\033[36m"
	white       = "\033[37m"
	bgBlue      = "\033[44m"
	bgRed       = "\033[41m"
	bgGreen     = "\033[42m"
	bgYellow    = "\033[43m"
	inverse     = "\033[7m"
	hideCursor  = "\033[?25l"
	showCursor  = "\033[?25h"
)

var oldState *term.State

func enableRawMode() {
	var err error
	oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable raw mode: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(hideCursor)
}

func disableRawMode() {
	fmt.Print(showCursor)
	fmt.Print(clearScreen)
	fmt.Printf(moveTo, 1, 1)
	if oldState != nil {
		term.Restore(int(os.Stdin.Fd()), oldState)
	}
}

func getTerminalSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

func readKey() byte {
	buf := make([]byte, 3)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return 0
	}

	// Handle escape sequences (arrow keys)
	if n == 3 && buf[0] == 27 && buf[1] == 91 {
		switch buf[2] {
		case 65: // Up
			return 'k'
		case 66: // Down
			return 'j'
		}
	}

	return buf[0]
}

// padRight pads a string to a fixed width, truncating if necessary.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padLeft pads a string to a fixed width with leading spaces.
func padLeft(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return strings.Repeat(" ", width-len(s)) + s
}
