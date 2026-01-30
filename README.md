# claudewatch

A comprehensive TUI monitor for Claude Code CLI instances on macOS. View real-time CPU/memory metrics, session history, complete conversation messages, and more—all in an interactive terminal interface.

**Monitor your Claude coding sessions with ease.** `claudewatch` gives you instant visibility into all running Claude instances, their working directories, session history, and full conversation transcripts with detailed message analytics.

<img alt="claudewatch demo" src="https://via.placeholder.com/600x400?text=claudewatch+TUI" width="600">

## Features

### Process Monitoring
- **Real-time metrics** – CPU usage, memory consumption, uptime
- **Working directory tracking** – See which project each Claude instance is working in (via macOS `proc_pidinfo`)
- **Process filtering** – Toggle MCP helper processes visibility
- **Color-coded alerts** – Visual indicators for high CPU/memory usage

### Session Management
- **Session discovery** – Automatically find all Claude sessions from `~/.claude/projects/`
- **Last message display** – See the timestamp and preview of the last message in each session
- **Responsive sorting** – Sessions sorted by last activity (newest first)
- **Sortable metadata** – Version, git branch, token usage, session duration
- **Sidechain indication** – Quickly identify branched conversations

### Message Viewing & Analysis
- **Complete conversation history** – View all messages from any session
- **Message filtering** – Show only your prompts, Claude's responses, or both
- **Detailed analytics** – For each message see:
  - Message ID and timestamp
  - Model used (Claude version)
  - Token counts (input, output, cache reads/writes)
  - Estimated cost (based on current Claude API pricing)
  - Input/output ratio
  - Cache savings

- **Tool information** – Prominently display tools called by Claude with arguments
- **Type-specific formatting** – Different layouts for user prompts, assistant responses, and tool calls
- **Smart text wrapping** – Word-based wrapping preserves readability

### User Experience
- **Responsive navigation** – Vim-like keybindings with arrow key alternatives
- **Dynamic column sizing** – Table adapts to terminal width
- **Consistent sorting** – Message sort order maintained when opening detail view
- **Stable scrolling** – Delta-based viewport updates for smooth cursor tracking

## Installation

### Prerequisites

- macOS 10.14 or later (for working directory detection via `proc_pidinfo`)
- Go 1.23 or later
- Xcode Command Line Tools (for CGo compilation)

### Build from source

```bash
git clone https://github.com/thieso2/claudewatch.git
cd claudewatch
go build -o bin/claudewatch ./cmd/claudewatch
```

### Install globally

```bash
go install github.com/thieso2/claudewatch/cmd/claudewatch@latest
```

This installs the binary to `$GOPATH/bin/claudewatch` (typically `~/go/bin/claudewatch`).

Add to your PATH if not already there:
```bash
export PATH="$HOME/go/bin:$PATH"
```

## Quick Start

```bash
# Start monitoring Claude instances
claudewatch

# Monitor with custom refresh interval
claudewatch --interval 500ms

# Show all processes including MCP helpers
claudewatch --show-helpers
```

Press `q` or `Ctrl+C` to quit.

## Usage Guide

### View Modes

**Process View** (main screen)
- Shows all running Claude instances with real-time metrics
- Press `↑/↓` to navigate, `enter` to select a process

**Session View**
- Shows all sessions in the selected process's working directory
- Sorted by last message timestamp (newest first)
- Press `enter` to open a session's conversation

**Session Detail View**
- Displays all messages in the session as compact cards
- Each card shows: role, timestamp, content preview, metrics
- Press `↑/↓` to navigate, `enter` to see full message details

**Message Detail View**
- Full message content with complete analytics
- Type-specific formatting (user prompts vs. assistant responses vs. tool calls)
- Press `esc` to return to session view

### Keyboard Shortcuts

#### Navigation
| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `enter` | Open/select current item |
| `esc` | Go back to previous view |
| `q` / `Ctrl+C` | Quit application |

#### Process View
| Key | Action |
|-----|--------|
| `r` | Manual refresh |
| `f` | Toggle MCP helper visibility |

#### Message Filtering (Session Detail View)
| Key | Action |
|-----|--------|
| `u` | Show user prompts only |
| `a` | Show Claude responses only |
| `b` | Show all messages |
| `s` | Toggle message sort order (newest/oldest first) |

### Command-line Options

```bash
claudewatch [flags]

Flags:
  -interval duration
        Refresh interval for metrics (default "1s")
  -show-helpers
        Show MCP helper processes (default false)
```

### Examples

```bash
# Monitor with 2-second refresh
claudewatch --interval 2s

# View all processes including helpers with faster refresh
claudewatch --interval 500ms --show-helpers

# Standard monitoring
claudewatch
```

## Display Columns

### Process View
- **PID** – Process ID
- **CPU%** – CPU usage percentage (color-coded: green < 50%, yellow < 80%, red ≥ 80%)
- **MEM** – Memory usage in MB or GB
- **UPTIME** – Process runtime (e.g., "2h34m" or "45m")
- **WORKDIR** – Current working directory (truncated, ~ for home)
- **COMMAND** – Full command line

### Session View
- **VER** – Claude version (e.g., v2.1.25)
- **BRANCH** – Git branch when session was created
- **LAST MSG** – Timestamp of last message (YYYY-MM-DD HH:MM)
- **TOKENS** – Input/Output token counts (input/output)
- **START** – Session start time
- **LEN** – Session duration (e.g., "12h34m" or "45m")
- **PREVIEW** – Last message preview (truncated, max 50 chars)

### Session Detail View (Message Cards)
Each message card shows 4 lines:
1. **Header** – Role emoji, timestamp, model, message ID (8 chars)
2. **Content** – Message text preview (truncated, newlines collapsed)
3. **Metrics** – Token counts, cost estimate
4. **Separator** – Visual divider (bright for selected message)

## Message Analytics

When viewing a message in detail, `claudewatch` displays comprehensive analytics:

### User Messages
- Timestamp of when you sent the prompt
- Content with proper text wrapping

### Claude Responses
- **Model** – Which Claude version generated the response
- **Tokens** – Input tokens used (from your prompt) and output tokens generated
- **Cache** – Cache creation tokens (for future cache hits) and cache read tokens
- **Cost** – Estimated cost based on current Claude API pricing:
  - Input: $3 per 1M tokens
  - Cache read: $0.30 per 1M tokens (90% savings)
  - Output: $15 per 1M tokens
  - Cache creation: $3 per 1M tokens (counted toward cache)
- **Ratio** – Input/output token ratio
- **Savings** – Estimated cost savings from cache hits vs. full price

### Tool Calls
- **Tool name** – Which tool Claude attempted to use
- **Arguments** – Full tool arguments/parameters

## Architecture

### Directory Structure

```
claudewatch/
├── cmd/claudewatch/
│   └── main.go                      # Entry point, CLI flags
├── internal/
│   ├── monitor/
│   │   ├── process.go               # Process discovery & filtering
│   │   ├── metrics.go               # CPU/memory collection
│   │   ├── session_parser.go        # Session JSONL parsing
│   │   └── workdir_darwin.go        # macOS proc_pidinfo wrapper
│   ├── ui/
│   │   ├── model.go                 # Bubbletea state, data structures
│   │   ├── update.go                # Event handling, business logic
│   │   ├── view.go                  # Rendering, formatting
│   │   └── table.go                 # Table configuration, column widths
│   └── types/
│       └── process.go               # ClaudeProcess, SessionInfo types
├── go.mod
├── go.sum
├── LICENSE                          # MIT license
└── README.md
```

### Technology Stack

- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** – Elm-inspired TUI framework
- **[bubble-table](https://github.com/evertras/bubble-table)** – Sortable table component
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** – Terminal styling and layout
- **[gopsutil v4](https://github.com/shirou/gopsutil)** – System metrics collection
- **CGo** – System-level working directory detection (macOS only)

## How It Works

### Process Detection

Claude CLI instances are identified by:
1. Executable path contains "claude"
2. Located in `/opt/homebrew/*/claude`
3. NOT the desktop app (`Claude.app`)

MCP helper processes (identified by `--claude-in-chrome-mcp` flag) can be toggled with `f` key.

### Working Directory Discovery

On macOS, `claudewatch` uses CGo to call the native `proc_pidinfo` system call:
```c
proc_pidinfo(pid, PROC_PIDVNODEPATHINFO, ...)
```

This retrieves the current working directory directly from the kernel, which is:
- More reliable than reading `/proc` (not available on macOS)
- More accurate than reading environment variables
- Shows "[Permission Denied]" gracefully if access is denied

### Session Discovery

Sessions are found in `~/.claude/projects/[encoded-path]/` where:
- Each `.jsonl` file is one session
- Files contain structured message history with metadata
- Sessions are automatically parsed and sorted by last activity

### Message Parsing

Each session's `.jsonl` file is parsed line-by-line with a 512KB initial buffer (up to 10MB max) to handle large conversations:
- Messages include role, content, timestamp, tokens, model info
- Tool calls are parsed with full argument information
- Cache hits/creation tracked separately
- Message cost calculated in real-time

## Building for Distribution

```bash
# Standard build
go build -o bin/claudewatch ./cmd/claudewatch

# Optimized build (smaller binary)
go build -ldflags="-s -w" -o bin/claudewatch ./cmd/claudewatch

# Build with version info
VERSION=$(git describe --tags --always)
go build -ldflags="-s -w -X main.Version=$VERSION" -o bin/claudewatch ./cmd/claudewatch
```

## Troubleshooting

### Build fails with CGo errors
Ensure Xcode Command Line Tools are installed:
```bash
xcode-select --install
```

### No Claude processes appear
1. Verify Claude CLI is running: `ps aux | grep claude`
2. Try manual refresh with `r` key
3. Check with `claudewatch --show-helpers` to see all processes
4. Look at the footer message for any errors

### Permission denied for working directory
Some processes may not allow directory access (e.g., processes from other users). This is expected and displays as "[Permission Denied]".

### Session files not loading
- Check that `~/.claude/projects/` exists and is readable
- Ensure session `.jsonl` files are valid (not corrupted)
- Look for error messages in the session view footer

### Memory usage or slowness with large sessions
`claudewatch` uses fixed-height message cards (4 lines each) for efficient rendering, even with thousands of messages. If you experience slowness:
1. Try increasing refresh interval: `claudewatch --interval 2s`
2. Filter messages to reduce visible count: `u` for prompts only
3. Ensure terminal has sufficient scrollback buffer

## Performance Characteristics

- **Process monitoring** – Negligible overhead, minimal system calls
- **Session parsing** – First load of large sessions may take 1-2s
- **Message filtering** – Instant (in-memory filter)
- **Rendering** – Optimized with delta-based viewport updates
- **Memory** – Typical usage 20-40MB for hundreds of sessions

## Future Enhancements

- [ ] Linux support with `/proc/[pid]/cwd` alternative
- [ ] Windows support with `GetCurrentDirectoryForProcess` API
- [ ] Historical metrics graphs
- [ ] Process tree view (parent-child relationships)
- [ ] Export session to markdown/PDF
- [ ] Alert on process crash or token limit exceeded
- [ ] Configuration file (~/.claudewatchrc)
- [ ] Custom color schemes
- [ ] Search/filter by keywords in messages

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss) from Charm
- Process metrics powered by [gopsutil](https://github.com/shirou/gopsutil)
- Inspired by tools like `top`, `htop`, and `watch`

---

**Made with ❤️ for Claude Code enthusiasts**
