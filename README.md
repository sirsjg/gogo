# gogo

Minimal CLI for streaming LLM responses. One prompt in, tokens out. No chat history, no frills.

## Install

```sh
brew install sirsjg/gogo
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

## I/O Contract

- **stdout**: LLM output only (machine-consumable)
- **stderr**: diagnostics, errors, logs (human-readable)
