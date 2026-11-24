package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/SeanMcP/ste-check/ste"
)

//go:embed schema.json
var schema []byte

const starterConfig = `{
  "$schema": "./schema.json",
  "type": "procedure",
  "maxWords": { "procedure": 20, "description": 25 },
  "rules": {
    "max-sentence-length": true,
    "no-unapproved-words": true,
    "no-ing": true,
    "no-passive": true,
    "no-contractions": true,
    "no-perfect-tense": true
  },
  "words": {},
  "allowIng": [],
  "ignore": []
}
`

type config struct {
	Schema   string               `json:"$schema"`
	Type     ste.TextType         `json:"type"`
	MaxWords map[ste.TextType]int `json:"maxWords"`
	Rules    map[ste.Rule]bool    `json:"rules"`
	Words    map[string]string    `json:"words"`
	AllowIng []string             `json:"allowIng"`
	Ignore   []string             `json:"ignore"`
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "ste-check", "config.json")
}

// loadConfig reads a config file. A missing file is an error only when the
// user named it with --config.
func loadConfig(path string, explicit bool) (config, error) {
	var c config
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) && !explicit {
		return c, nil
	}
	if err != nil {
		return c, err
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			line := bytes.Count(data[:syntaxErr.Offset], []byte("\n")) + 1
			return c, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if err := c.validate(); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func (c config) validate() error {
	types := []ste.TextType{ste.Procedure, ste.Description}
	if c.Type != "" && !slices.Contains(types, c.Type) {
		return fmt.Errorf("type must be %q or %q, not %q", ste.Procedure, ste.Description, c.Type)
	}
	for t, n := range c.MaxWords {
		if !slices.Contains(types, t) {
			return fmt.Errorf("maxWords: unknown text type %q", t)
		}
		if n < 1 {
			return fmt.Errorf("maxWords.%s must be 1 or more", t)
		}
	}
	for r := range c.Rules {
		if !slices.Contains(ste.Rules, r) {
			return fmt.Errorf("rules: unknown rule %q. The rules are: %s", r, joinRules())
		}
	}
	for bad, good := range c.Words {
		if strings.TrimSpace(bad) == "" || strings.TrimSpace(good) == "" {
			return fmt.Errorf("words: %q = %q: the word and its alternative must not be empty", bad, good)
		}
	}
	return nil
}

func (c config) options() ste.Options {
	disabled := map[ste.Rule]bool{}
	for r, on := range c.Rules {
		disabled[r] = !on
	}
	return ste.Options{
		MaxWords: c.MaxWords,
		Disabled: disabled,
		Words:    c.Words,
		AllowIng: c.AllowIng,
		Ignore:   c.Ignore,
	}
}

// initConfig writes a starter config.json (if there is none) and schema.json
// (always, so it matches this version) to the config file's directory.
func initConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	schemaPath := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(schemaPath, schema, 0o644); err != nil {
		return err
	}
	fmt.Println("Wrote", schemaPath)

	if _, err := os.Stat(path); err == nil {
		fmt.Println("Kept", path, "(it already exists)")
		return nil
	}
	if err := os.WriteFile(path, []byte(starterConfig), 0o644); err != nil {
		return err
	}
	fmt.Println("Wrote", path)
	return nil
}

func joinRules() string {
	names := make([]string, len(ste.Rules))
	for i, r := range ste.Rules {
		names[i] = string(r)
	}
	return strings.Join(names, ", ")
}
