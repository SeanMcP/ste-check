package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/SeanMcP/ste-check/ste"
)

func writeConfig(t *testing.T, content string) string {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSchemaMatchesRules(t *testing.T) {
	var s struct {
		Properties struct {
			Rules struct {
				Properties map[ste.Rule]any `json:"properties"`
			} `json:"rules"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		t.Fatal(err)
	}
	var got []ste.Rule
	for r := range s.Properties.Rules.Properties {
		got = append(got, r)
	}
	slices.Sort(got)
	want := slices.Sorted(slices.Values(ste.Rules))
	if !slices.Equal(got, want) {
		t.Errorf("schema rules %v do not match ste.Rules %v", got, want)
	}
}

func TestStarterConfigLoads(t *testing.T) {
	if _, err := loadConfig(writeConfig(t, starterConfig), true); err != nil {
		t.Error(err)
	}
}

func TestLoadConfig(t *testing.T) {
	cfg, err := loadConfig(writeConfig(t, `{"type": "description", "rules": {"no-contractions": false}}`), true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Type != ste.Description || !cfg.options().Disabled[ste.Contraction] {
		t.Errorf("got %+v", cfg)
	}
}

func TestConfigErrors(t *testing.T) {
	cases := map[string]string{
		"unknown key":  `{"rulez": {}}`,
		"unknown rule": `{"rules": {"no-contraction": false}}`,
		"bad type":     `{"type": "manual"}`,
		"bad maxWords": `{"maxWords": {"procedure": 0}}`,
		"empty word":   `{"words": {"leverage": ""}}`,
	}
	for name, content := range cases {
		if _, err := loadConfig(writeConfig(t, content), true); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestMissingConfig(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "config.json")
	if _, err := loadConfig(missing, false); err != nil {
		t.Errorf("default path: want no error, got %v", err)
	}
	if _, err := loadConfig(missing, true); err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Errorf("--config path: want a not-found error, got %v", err)
	}
}
