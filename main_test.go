package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

func TestBuildPromptIncludesCLIEnvironment(t *testing.T) {
	prompt := buildPrompt("list files", cliEnvironment{
		OS: "darwin", Architecture: "arm64", Shell: "zsh",
		ShellVersion: "5.9", Terminal: "xterm-256color",
	})
	for _, want := range []string{
		"- OS: darwin", "- Architecture: arm64", "- Shell: zsh",
		"- Shell version: 5.9", "- Terminal type: xterm-256color",
		`User request: "list files"`,
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt does not contain %q:\n%s", want, prompt)
		}
	}
}

func TestLocalCLIEnvironmentUsesActiveShellMetadata(t *testing.T) {
	t.Setenv("HOWDO_SHELL", "bash")
	t.Setenv("HOWDO_SHELL_VERSION", "5.2.37")
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("TERM", "screen")
	env := localCLIEnvironment()
	if env.Shell != "bash" || env.ShellVersion != "5.2.37" || env.Terminal != "screen" {
		t.Fatalf("localCLIEnvironment() = %+v", env)
	}
}

func TestShellWrappersPassActiveShellMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, executable, wrapper, shell, command string
	}{
		{"bash", "bash", "shell/howdo.bash", "bash", `source "$HOWDO_WRAPPER"; howdo list files`},
		{"bash-readline", "bash", "shell/howdo.bash", "bash", `source "$HOWDO_WRAPPER"; READLINE_LINE="list files"; __howdo_readline`},
		{"zsh", "zsh", "shell/howdo.zsh", "zsh", `source "$HOWDO_WRAPPER"; howdo list files`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := exec.LookPath(tc.executable); err != nil {
				t.Skipf("%s is not installed", tc.executable)
			}
			tmp := t.TempDir()
			capture := filepath.Join(tmp, "capture")
			fake := filepath.Join(tmp, "howdo-fake")
			script := "#!/bin/sh\n" +
				"printf '%s\\n' \"$HOWDO_SHELL\" \"$HOWDO_SHELL_VERSION\" \"$*\" > \"$HOWDO_CAPTURE\"\n" +
				"printf '%s\\n' \"HOWDO_COMMAND='echo ok'\" \"HOWDO_RATIONALE='compatible'\"\n"
			if err := os.WriteFile(fake, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			wrapper, err := filepath.Abs(tc.wrapper)
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(tc.executable, "-c", tc.command)
			cmd.Env = append(os.Environ(), "HOWDO_BIN="+fake, "HOWDO_CAPTURE="+capture, "HOWDO_WRAPPER="+wrapper)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("wrapper failed: %v\n%s", err, output)
			}
			got, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(got)), "\n")
			if len(lines) != 3 || lines[0] != tc.shell || lines[1] == "" || lines[2] != "--shell list files" {
				t.Fatalf("captured metadata = %q", got)
			}
		})
	}
}
