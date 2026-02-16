# ReleaseRadar

A terminal UI for tracking GitHub releases across multiple repositories, with AI-powered summaries.

![AI Chat](screenshots/chat.gif)

## Features

- Track releases from multiple GitHub repositories
- Split-pane TUI with releases table + detail view
- Repository info panel with stars, forks, language, topics
- Changelog parsing fallback for repos without GitHub Releases
- AI-powered release summaries (OpenAI)
- Interactive AI chat about your tracked releases
- Add/remove repos from within the TUI or via CLI

## Installation

### Homebrew

```bash
brew install wdm0006/tap/releaseradar
```

### From source

```bash
go install github.com/wdm0006/releaseradar/cmd/releaseradar@latest
```

## Prerequisites

- [GitHub CLI](https://cli.github.com) (`gh`) installed and authenticated
- `OPENAI_API_KEY` environment variable (optional, for AI features)

## Usage

```bash
# Launch the TUI
releaseradar

# Add a repository
releaseradar add owner/repo

# Remove a repository
releaseradar remove owner/repo

# List tracked repositories
releaseradar list
```

## TUI Key Bindings

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `r` | Refresh releases |
| `a` | Add repository |
| `d` / `x` | Remove selected repository |
| `s` | Generate AI summary (on Summary tab) |
| `Ctrl+Left/Right` | Switch tabs |
| `Tab` | Toggle focus between panels |

## Configuration

Tracked repositories are stored in `~/.releaseradar.json`.

## License

MIT
