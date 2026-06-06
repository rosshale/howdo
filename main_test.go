package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseSuggestion(t *testing.T) {
	raw := []byte(`{"command":" df -h ","rationale":" Shows disk usage. "}`)
	got, err := parseSuggestion(raw)
	if err != nil {
		t.Fatalf("parseSuggestion returned error: %v", err)
	}
	if got.Command != "df -h" {
		t.Fatalf("Command = %q, want %q", got.Command, "df -h")
	}
	if got.Rationale != "Shows disk usage." {
		t.Fatalf("Rationale = %q, want %q", got.Rationale, "Shows disk usage.")
	}
}

func TestParseSuggestionRejectsEmptyCommand(t *testing.T) {
	_, err := parseSuggestion([]byte(`{"command":" ","rationale":"No command."}`))
	if err == nil {
		t.Fatal("parseSuggestion succeeded, want error")
	}
	if !strings.Contains(err.Error(), "empty command") {
		t.Fatalf("error = %q, want empty command error", err)
	}
}

func TestParseSuggestionRejectsMultilineCommand(t *testing.T) {
	_, err := parseSuggestion([]byte("{\"command\":\"echo one\\necho two\",\"rationale\":\"Two commands.\"}"))
	if err == nil {
		t.Fatal("parseSuggestion succeeded, want error")
	}
	if !strings.Contains(err.Error(), "multi-line") {
		t.Fatalf("error = %q, want multi-line error", err)
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote("printf '%s\\n' hello")
	want := "'printf '\\''%s\\n'\\'' hello'"
	if got != want {
		t.Fatalf("shellQuote = %q, want %q", got, want)
	}
}

func TestRunRejectsConflictingOutputModes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"--json", "--zsh", "list files"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if err == nil {
		t.Fatal("err = nil, want error")
	}
}

func TestRunRejectsConflictingShellOutputMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"--json", "--shell", "list files"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if err == nil {
		t.Fatal("err = nil, want error")
	}
}

func TestJSONOutputShape(t *testing.T) {
	s := suggestion{Command: "df -h", Rationale: "Shows disk usage."}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	if err := enc.Encode(s); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"command":"df -h"`) {
		t.Fatalf("encoded output = %q", out.String())
	}
}
