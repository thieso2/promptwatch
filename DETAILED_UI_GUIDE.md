# claudewatch Detailed UI Information Guide

This document describes all the detailed information now displayed throughout the claudewatch UI.

---

## Session List View

The session table now displays comprehensive metadata about each session.

### Columns

| Column | Header | Width | Content |
|--------|--------|-------|---------|
| 1 | VER | 6 chars | Claude version (`v2.1.1`, `-` if unknown) |
| 2 | BRANCH | 10 chars | Git branch active (`main`, `feature/x`, etc.) |
| 3 | TOKENS | 12 chars | Token usage `input/output` (e.g., `1234/567`) |
| 4 | START | 12 chars | Session start time (`MM-DD HH:MM`) |
| 5 | LEN | 7 chars | Duration (`12h34m`, `45m`) |
| 6 | USR | 5 chars | User prompt count |
| 7 | INT | 3 chars | Resumptions (>1 hour gaps) |
| 8 | TITLE | Remaining | Session title with indicators |

### Example Row

```
v2.1.1  |  main      |  1234/567    |  01-30 14:23  |  2h30m  |  12  |  0  |  My Project Session
```

### Special Indicators

- **🔀 Prefix**: Indicates side-chain (branching conversation)
- **"-"**: Shows when data is unavailable
- **Input/Output Format**: Shows balance of conversation input vs Claude's response

---

## Session Detail View

Comprehensive view of a single session with all available metadata.

### Header Section (Top)

#### Line 1: Title
```
Session Details
```

#### Line 2: Path
```
Path: ~/Projects/MyProject/.claude/projects/...
```

#### Line 3: Metadata
```
v:2.1.1  |  branch:main  |  🔀side-chain  |  tokens:5000→2000  |  prompts:12  |  resumptions:1
```

Displays:
- **v:X.X.X** - Claude version from JSONL entries
- **branch:BRANCH** - Git branch when session was created
- **🔀side-chain** - Only if this is a branching conversation
- **tokens:input→output** - Total tokens used
- **prompts:N** - How many user messages
- **resumptions:N** - Times session was resumed (>1 hour gaps)

#### Line 4: First Prompt Preview
```
Initial: create a comprehensive README.md in here and link all...
```

Shows first 80 characters of the initial user prompt that started the session.

### Stats Section (Middle)

#### Summary Line
```
Messages: 15 (9 user, 6 assistant) | Duration: 2h34m | Created: 2026-01-30 14:00 | Updated: 2026-01-30 16:34
```

#### Detailed Stats
```
Total: 15  |  User Prompts: 9  |  Claude Responses: 6  |  Duration: 2h34m
Tokens: Input 5000, Output 2000, Cache-Create 1000, Cache-Hit 500
```

### Messages Section

Displays filtered message table based on filter mode:
- **All**: All user and assistant messages
- **User**: Only user prompts
- **Claude**: Only assistant responses

Filter status shown with count:
```
Messages: [User Prompts: 9]  or  [Claude Responses: 6]  or  [All Messages: 15]
```

---

## Message Detail View

Complete information about a single message with token analysis.

### Header Information

#### Message Title
```
Claude Response
Your Prompt
Claude Tool Call: write_file
Tool Result: write_file
```

Shows message type based on content.

#### Timestamp
```
Time: 2026-01-30 14:23:45
```

#### Tool Information (if applicable)
```
Tool: write_file | Arguments: {"path": "README.md", "content": "..."}
```

#### Token Usage Information (Assistant Responses Only)
```
Model: claude-sonnet-4-5-20250929 | Tokens: 1234 → 567 | Cache-Create: 100 | Cache-Hit: 50 | Approx: $0.021456
```

**Breakdown:**
- **Model**: Exact Claude model that generated response
- **Tokens: INPUT → OUTPUT**: 
  - INPUT: Tokens from the prompt/context
  - OUTPUT: Tokens in Claude's response
- **Cache-Create**: Tokens written to prompt cache (creates cached tokens for future reuse)
- **Cache-Hit**: Tokens read from existing cache (cheaper than fresh tokens)
- **Approx Cost**: Estimated API cost for this message
  - Input tokens: $3 per 1M
  - Cache creation: $3 per 1M  
  - Cache hits: $0.30 per 1M (90% savings)
  - Output tokens: $15 per 1M

### Message Content

Full message text with:
- Line wrapping to terminal width
- Scrolling support (↑/↓ keys)
- Pagination indicator: `Line X-Y of Z`

### Navigation

```
↑/↓: Scroll  |  ←/→: Prev/Next Message  |  PgUp/PgDn: Page  |  Home/End: Jump  |  esc: Back  |  q: Quit
```

---

## Information Captured

### Per Session
- **Metadata**: Version, git branch, created/modified times
- **Duration**: From first to last message
- **Message Counts**: Total, user, assistant
- **Token Usage**: Input, output, cache statistics
- **Structure**: Resumption count, side-chain status
- **Content**: First prompt, all messages

### Per Message
- **Identity**: Role (user/assistant), timestamp, type
- **Content**: Full text, tool name/arguments if applicable
- **Tokens** (assistant only): Model, input/output, cache info
- **Timing**: Exact ISO 8601 timestamp

### Metrics
- **Cost Estimation**: Based on actual token counts and current pricing
- **Cache Efficiency**: Shows tokens saved by caching
- **Duration**: Total session time and gap-based resumption detection
- **Interruptions**: Sessions resumed after >1 hour gaps

---

## Color Coding

| Color | Meaning |
|-------|---------|
| Yellow (11) | Titles, important information |
| Cyan (6) | Token/cost information |
| Green (10) | Statistics, positive metrics |
| Gray (8) | Secondary info, timestamps |
| Red (1) | Empty filter results |

---

## Usage Examples

### Understanding Token Usage

If you see:
```
Model: claude-sonnet-4-5-20250929 | Tokens: 1234 → 567 | Cache-Hit: 200 | Approx: $0.012345
```

This means:
- 1,234 tokens were input (context + your prompt)
- 567 tokens were Claude's response
- 200 of the input tokens came from cache (faster, cheaper)
- This message cost approximately $0.012345

### Identifying Expensive Sessions

In session list, look for high TOKENS values:
```
v2.1.1  |  main  |  50000/25000  |  ...  |  tokens:50000→25000
```

This session used 50k input and 25k output tokens - expensive!

### Tracking Session Progress

The metadata line shows:
```
tokens:10000→5000  |  prompts:25  |  resumptions:3
```

Means:
- Used 10k input tokens, 5k output (1:2 ratio, lot of input)
- 25 user messages submitted
- Session was paused and resumed 3 times (likely over multiple days)

---

## Tips

1. **Check Cache Efficiency**: Higher cache-hit counts mean more cost savings
2. **Monitor Token Balance**: Input-heavy sessions (many small messages) vs output-heavy (long responses)
3. **Session Duration**: Helps identify multi-day sessions vs single-session work
4. **Branch Context**: Shows which code branches were active (useful for architectural changes)
5. **Side-chain Indicator**: Branching conversations useful for alternative approaches

---

## Related Files

- [CLAUDE_SESSION.md](./CLAUDE_SESSION.md) - Format of session JSONL files
- [internal/monitor/session_parser.go](./internal/monitor/session_parser.go) - Code that extracts this data
- [internal/ui/view.go](./internal/ui/view.go) - Code that displays this information
