# gogo

[![GitHub release](https://img.shields.io/github/v/release/sirsjg/gogo)](https://github.com/sirsjg/gogo/releases) [![Go Report Card](https://goreportcard.com/badge/github.com/sirsjg/gogo)](https://goreportcard.com/report/github.com/sirsjg/gogo) [![Build Status](https://github.com/sirsjg/gogo/workflows/CI/badge.svg)](https://github.com/sirsjg/gogo/actions) ![Go Version](https://img.shields.io/github/go-mod/go-version/sirsjg/gogo) ![macOS](https://img.shields.io/badge/macOS-000000?style=flat&logo=apple&logoColor=white) ![Linux](https://img.shields.io/badge/Linux-FCC624?style=flat&logo=linux&logoColor=black) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Minimal CLI for streaming LLM responses. One prompt in, tokens out. No chat history, no frills.

## Install

```sh
brew tap sirsjg/gogo
brew install gogo
```

## Usage

```sh
gogo -P openai -p "Hello"
gogo -P anthropic < prompt.txt
cat file.go | gogo -P gemini -p "Review this code"
```

## Options

```
-p, --prompt <text>       Inline prompt (if empty, reads from stdin)
-P, --provider <name>     Provider: openai | anthropic | gemini
-m, --model <name>        Model name (provider-specific defaults)
-M, --max-tokens <n>      Maximum output tokens
-T, --temperature <n>     Sampling temperature (0.0 - 2.0)
-c, --config <path>       Path to config.json
-t, --timeout <duration>  Request timeout (e.g., 30s, 1m)
-d, --debug               Enable verbose stderr logging
-v, --version             Print version and exit
-u, --update              Check for updates
-h, --help                Show help message
```

## Configuration

**Priority**: flags > environment > config file > defaults

### Environment Variables

```sh
OPENAI_API_KEY       # OpenAI API key
ANTHROPIC_API_KEY    # Anthropic API key
GEMINI_API_KEY       # Google Gemini API key
GOGO_PROVIDER        # Default provider
GOGO_MODEL           # Default model
```

### Config File

Location: `~/.config/gogo/config.json`

```json
{
  "provider": "openai",
  "model": "gpt-4o-mini",
  "max_tokens": 512,
  "temperature": 0.2
}
```

## Tools

The LLM has access to a filesystem tool (`fs`) for local file operations:

| Operation | Description |
|-----------|-------------|
| `read`    | Read file contents |
| `write`   | Create or overwrite a file |
| `append`  | Append to a file |
| `delete`  | Remove file or directory recursively |
| `mkdir`   | Create directory (with parents) |
| `rmdir`   | Remove empty directory |
| `list`    | List directory contents |
| `stat`    | Get file/directory info |
| `move`    | Rename or move a path |
| `copy`    | Copy a file |

## I/O Contract

- **stdout**: LLM output only (machine-consumable)
- **stderr**: diagnostics, errors, logs (human-readable)
