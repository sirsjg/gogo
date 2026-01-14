package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"gogo/internal/config"
	"gogo/internal/prompt"
	"gogo/internal/provider"
)

// Version info injected by goreleaser ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, `gogo %s - streaming LLM CLI

Usage: gogo [options] [-p prompt | < input]

Options:
  -p, --prompt <text>       Inline prompt (if empty, reads from stdin)
  -P, --provider <name>     Provider: openai | anthropic | gemini
  -m, --model <name>        Model name (provider-specific defaults)
  -M, --max-tokens <n>      Maximum output tokens
  -T, --temperature <n>     Sampling temperature (0.0 - 2.0)
  -c, --config <path>       Path to config.json
  -t, --timeout <duration>  Request timeout (e.g., 30s, 1m)
  -d, --debug               Enable verbose stderr logging
  -v, --version             Print version and exit
  -h, --help                Show this help message

Examples:
  gogo -P openai -p "Hello"
  gogo -P anthropic < prompt.txt
  cat file.go | gogo -P gemini -p "Review this code"

Environment:
  OPENAI_API_KEY       OpenAI API key
  ANTHROPIC_API_KEY    Anthropic API key
  GEMINI_API_KEY       Google Gemini API key
  GOGO_PROVIDER        Default provider
  GOGO_MODEL           Default model

Config: ~/.config/gogo/config.json
`, version)
}

func main() {
	stderr := os.Stderr

	// Custom usage function
	flag.Usage = printUsage

	flags := config.Flags{}
	var showHelp bool

	// Short and long flag pairs
	flag.StringVar(&flags.Prompt, "p", "", "")
	flag.StringVar(&flags.Prompt, "prompt", "", "")
	flag.StringVar(&flags.Provider, "P", "", "")
	flag.StringVar(&flags.Provider, "provider", "", "")
	flag.StringVar(&flags.Model, "m", "", "")
	flag.StringVar(&flags.Model, "model", "", "")
	flag.IntVar(&flags.MaxTokens, "M", 0, "")
	flag.IntVar(&flags.MaxTokens, "max-tokens", 0, "")
	flag.Float64Var(&flags.Temperature, "T", 0, "")
	flag.Float64Var(&flags.Temperature, "temperature", 0, "")
	flag.StringVar(&flags.ConfigPath, "c", "", "")
	flag.StringVar(&flags.ConfigPath, "config", "", "")
	flag.DurationVar(&flags.Timeout, "t", 0, "")
	flag.DurationVar(&flags.Timeout, "timeout", 0, "")
	flag.BoolVar(&flags.Debug, "d", false, "")
	flag.BoolVar(&flags.Debug, "debug", false, "")
	flag.BoolVar(&flags.Version, "v", false, "")
	flag.BoolVar(&flags.Version, "version", false, "")
	flag.BoolVar(&showHelp, "h", false, "")
	flag.BoolVar(&showHelp, "help", false, "")
	flag.Parse()

	// Show help if requested or no arguments provided
	if showHelp || (len(os.Args) == 1 && !prompt.HasStdin()) {
		printUsage()
		os.Exit(0)
	}

	if flags.Version {
		fmt.Fprintln(stderr, "gogo", version)
		os.Exit(0)
	}

	cfg, err := config.Load(flags)
	if err != nil {
		fmt.Fprintln(stderr, "config error:", err)
		os.Exit(1)
	}

	promptText, err := prompt.Read(flags.Prompt)
	if err != nil {
		fmt.Fprintln(stderr, "prompt error:", err)
		os.Exit(1)
	}
	if promptText == "" {
		fmt.Fprintln(stderr, "prompt error: no prompt provided")
		os.Exit(1)
	}

	ctx := context.Background()
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	client := provider.NewClient(cfg, stderr)
	if err := client.Stream(ctx, promptText, os.Stdout); err != nil {
		fmt.Fprintln(stderr, "provider error:", err)
		os.Exit(1)
	}

	_ = os.Stdout.Sync()
}
