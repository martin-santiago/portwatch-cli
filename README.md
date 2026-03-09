# portwatch-cli

> Lightweight terminal UI to monitor listening ports and kill processes — single binary, zero dependencies, runs on macOS and Linux.

```
┌──────────────────────────────────────────────────────────┐
│  PortWatch CLI                                           │
│  4 ports listening                        FILTER ON      │
│                                                          │
│  PORT      PID  PROCESS               USER               │
│  ──────────────────────────────────────────────────       │
│  :3001     1234  node                  mprez              │
│▸ :3002     1235  node                  mprez              │
│  :5432      789  postgres              _postgres          │
│  :8080    45678  go                    mprez              │
│                                                          │
│──────────────────────────────────────────────────────────│
│  ↑/↓/j/k navigate  enter/x kill  f filter  q quit       │
└──────────────────────────────────────────────────────────┘
```

## Features

- **Interactive TUI** — navigate ports, kill processes, toggle filters — all from the terminal
- **Non-interactive mode** — pipe-friendly commands for scripting (`list`, `kill`, `filter`)
- **Cross-platform** — works on macOS (`lsof`) and Linux (`ss`/`netstat`)
- **Filter mode** — watch only the ports you care about (e.g., 3001, 3002, 8000)
- **Persistent config** — filter settings saved to `~/.portwatch.json`
- **Zero runtime dependencies** — single static binary, `CGO_ENABLED=0`
- **Tiny footprint** — ~2MB binary, minimal RAM usage

---

## Installation

### Option 1: Build from source (recommended)

Requires **Go 1.21+**.

```bash
# Install Go if needed
# macOS:
brew install go
# Linux (Debian/Ubuntu):
sudo apt install golang
# Linux (other): https://go.dev/dl/

# Clone, build, and install
git clone https://github.com/martin-santiago/portwatch-cli.git
cd portwatch-cli
make install
```

This installs the `pw` binary to:
- **Linux**: `~/.local/bin/pw` (no sudo needed)
- **macOS**: `/usr/local/bin/pw`

Verify it works:

```bash
pw version
```

### Option 2: One-liner install

```bash
git clone https://github.com/martin-santiago/portwatch-cli.git /tmp/portwatch-cli \
  && cd /tmp/portwatch-cli \
  && make install \
  && cd - \
  && rm -rf /tmp/portwatch-cli
```

### Option 3: Manual download

```bash
git clone https://github.com/martin-santiago/portwatch-cli.git
cd portwatch-cli
make build
# Binary is at build/pw — copy it wherever you want
cp build/pw ~/.local/bin/   # Linux
cp build/pw /usr/local/bin/ # macOS
```

> **Note (Linux):** Make sure `~/.local/bin` is in your `$PATH`. If not, add this to your `~/.bashrc` or `~/.zshrc`:
> ```bash
> export PATH="$HOME/.local/bin:$PATH"
> ```

---

## Quick Start

```bash
# Interactive TUI — just type:
pw

# Or use commands directly:
pw list          # show all listening ports
pw kill 3001     # kill everything on port 3001
pw filter        # show filter config
```

---

## Usage

### Interactive TUI (default)

```bash
pw
```

#### Keyboard shortcuts — Port list

| Key | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Navigate the port list |
| `enter` / `x` | Kill the selected process |
| `f` | Toggle filter mode ON/OFF |
| `e` | Edit filter list (add/remove ports) |
| `r` | Force refresh |
| `q` / `Ctrl+C` | Quit |

#### Keyboard shortcuts — Filter editor

| Key | Action |
|---|---|
| `↑` / `↓` / `j` / `k` | Navigate filter list |
| `a` | Add a new port |
| `d` / `x` | Remove selected port |
| `esc` | Back to port list |

### Non-interactive commands

Perfect for scripting, cron jobs, or quick one-liners:

```bash
# List all listening ports
pw list

# List only filtered ports
pw list --filter

# Kill everything on port 3001
pw kill 3001

# Kill a specific PID
pw kill 12345 --pid

# Show current filter config
pw filter

# Enable/disable filter mode
pw filter on
pw filter off
pw filter toggle

# Add/remove ports from watch list
pw filter add 4000
pw filter rm 4000

# Help
pw help
```

### One-liner examples

```bash
# Kill all node processes on your dev ports
pw list --filter | grep node | awk '{print $2}' | xargs kill

# Check if port 3001 is in use
pw list | grep :3001 && echo "Port 3001 is busy"

# List ports as a quick check (non-interactive)
pw ls
```

---

## Configuration

Stored at `~/.portwatch.json`:

```json
{
  "filter_ports": [3001, 3002, 3003, 3005, 7000, 8000],
  "filter_enabled": false,
  "refresh_interval_seconds": 3
}
```

| Field | Description | Default |
|---|---|---|
| `filter_ports` | Ports shown when filter mode is ON | `[3001, 3002, 3003, 3005, 7000, 8000]` |
| `filter_enabled` | Whether filter mode is active | `false` |
| `refresh_interval_seconds` | TUI auto-refresh interval | `3` |

Edit this file directly if you prefer — changes apply on the next refresh.

---

## How It Works

```
pw (no args)                     pw list|kill|filter
       │                                   │
       ▼                                   ▼
  Interactive TUI                  Non-interactive CLI
  ┌─────────────┐                  ┌─────────────┐
  │ terminal.go │ raw mode         │  main.go    │ stdout table
  │ ui.go       │ ANSI rendering   │             │ or action
  └──────┬──────┘                  └──────┬──────┘
         │                                │
         ▼                                ▼
    ┌──────────┐                    ┌──────────┐
    │ ports.go │ lsof / ss scan    │ ports.go │
    │config.go │ ~/.portwatch.json │config.go │
    └──────────┘                    └──────────┘
```

| File | Role |
|---|---|
| `main.go` | Entry point, CLI arg routing, interactive loop |
| `ports.go` | Port scanning (macOS: `lsof`, Linux: `ss`/`netstat`), kill, filter |
| `config.go` | Thread-safe config with JSON persistence |
| `terminal.go` | Raw terminal mode, ANSI escape codes, key input |
| `ui.go` | TUI rendering (port list, filter editor, add port dialog) |

---

## Makefile

| Command | Description |
|---|---|
| `make build` | Build binary to `build/pw` |
| `make run` | Build and run interactively |
| `make install` | Install globally (`~/.local/bin` on Linux, `/usr/local/bin` on macOS) |
| `make uninstall` | Remove from install path |
| `make clean` | Delete `build/` directory |

## Uninstall

```bash
make uninstall              # remove pw binary
rm ~/.portwatch.json        # remove config (optional)
```

## Related

- [**PortWatch**](https://github.com/martin-santiago/portwatch) — macOS menu bar version (native Cocoa UI)

## License

MIT
