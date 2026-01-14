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

The LLM can use the `fs` tool for local file operations: `read`, `write`, `append`, `delete`, `mkdir`, `rmdir`, `list`, `stat`, `move`, `copy`.

### Custom Plugins

Add your own tools via `~/.config/gogo/plugins.json`:

```json
{
  "tools": [
    {
      "name": "weather",
      "description": "Get current weather for a location",
      "type": "http",
      "url": "https://api.example.com/weather?city={{.location}}",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer $API_KEY"
      },
      "input_schema": {
        "type": "object",
        "properties": {
          "location": {"type": "string", "description": "City name"}
        },
        "required": ["location"]
      }
    },
    {
      "name": "run-script",
      "description": "Execute a shell script",
      "type": "exec",
      "command": "/bin/bash",
      "args": ["-c", "{{.script}}"],
      "input_schema": {
        "type": "object",
        "properties": {
          "script": {"type": "string", "description": "Script to run"}
        },
        "required": ["script"]
      }
    }
  ]
}
```

**Plugin Types:**
- `http`: Make HTTP/API calls with templated URLs, headers, and bodies
- `exec`: Execute local commands with templated arguments

**Template Variables:**
- `{{.field}}` - Substitutes input field values
- `$ENV_VAR` - Substitutes environment variables (in URLs and headers)

See `examples/plugins.json` for more examples.

## I/O Contract

- **stdout**: LLM output only (machine-consumable)
- **stderr**: diagnostics, errors, logs (human-readable)
