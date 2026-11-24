package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SeanMcP/ste-check/ste"
)

const usage = `Check text for Simplified Technical English (unofficial).

Usage:
  ste-check [flags] "text to check"
  echo "text to check" | ste-check [flags]

Flags:
  -p, --procedure      Check as a procedure (the default)
  -d, --description    Check as a description
  -c, --config <path>  Use this config file (default: ~/.config/ste-check/config.json)
      --init           Write a starter config and its schema to the config directory
  -h, --help           Show this help

Exit codes: 0 no issues, 1 issues found, 2 usage or config error.
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var textType ste.TextType
	configPath, explicitConfig, doInit := defaultConfigPath(), false, false
	var texts []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if value, ok := strings.CutPrefix(arg, "--config="); ok {
			configPath, explicitConfig = value, true
			continue
		}
		switch arg {
		case "-p", "--procedure":
			textType = ste.Procedure
		case "-d", "--description":
			textType = ste.Description
		case "-c", "--config":
			if i+1 >= len(args) {
				return usageError(arg + " needs a path")
			}
			i++
			configPath, explicitConfig = args[i], true
		case "--init":
			doInit = true
		case "-h", "--help":
			fmt.Print(usage)
			return 0
		case "--":
			texts = append(texts, args[i+1:]...)
			i = len(args)
		default:
			if strings.HasPrefix(arg, "-") {
				return usageError("Unknown flag: " + arg)
			}
			texts = append(texts, arg)
		}
	}

	if doInit {
		if err := initConfig(configPath); err != nil {
			fmt.Fprintln(os.Stderr, "Cannot write config:", err)
			return 2
		}
		return 0
	}

	cfg, err := loadConfig(configPath, explicitConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Config error:", err)
		return 2
	}
	if textType == "" {
		textType = cfg.Type
	}
	if textType == "" {
		textType = ste.Procedure
	}

	text := strings.Join(texts, " ")
	if len(texts) == 0 {
		if isTerminal(os.Stdin) {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Cannot read stdin:", err)
			return 2
		}
		text = string(input)
	}

	checker := ste.New(cfg.options())
	sentences := checker.Check(text, textType)
	if len(sentences) == 0 {
		fmt.Fprintln(os.Stderr, "No text to check.")
		return 2
	}

	color := isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	if report(os.Stdout, sentences, textType, checker.MaxWords(textType), color) > 0 {
		return 1
	}
	return 0
}

func usageError(msg string) int {
	fmt.Fprintf(os.Stderr, "%s\n\n%s", msg, usage)
	return 2
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
