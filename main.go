package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

const codexPrompt = `You generate one shell command for a user's natural-language request.

Return only JSON matching the provided schema. Do not use markdown.

Rules:
- Produce exactly one command string that the user can review and run in zsh.
- Do not execute commands, inspect files, or use tools.
- Prefer common, portable CLI commands when possible.
- If the request is ambiguous, choose the safest reasonable command and mention the assumption in the rationale.
- The command must not include a trailing newline.

User request: %q`

const outputSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["command", "rationale"],
  "properties": {
    "command": {
      "type": "string",
      "minLength": 1
    },
    "rationale": {
      "type": "string",
      "minLength": 1
    }
  }
}`

type suggestion struct {
	Command   string `json:"command"`
	Rationale string `json:"rationale"`
}

type options struct {
	jsonOutput bool
	zshOutput  bool
	noSpinner  bool
	codexPath  string
}

func main() {
	code, err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "howdo: %v\n", err)
	}
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) (int, error) {
	var opts options
	flags := flag.NewFlagSet("howdo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.BoolVar(&opts.jsonOutput, "json", false, "print machine-readable JSON only")
	flags.BoolVar(&opts.zshOutput, "zsh", false, "print zsh assignment statements for shell integration")
	flags.BoolVar(&opts.noSpinner, "no-spinner", false, "disable progress spinner")
	flags.StringVar(&opts.codexPath, "codex", getenv("HOWDO_CODEX", "codex"), "path to the codex executable")

	if err := flags.Parse(args); err != nil {
		return 2, err
	}
	if opts.jsonOutput && opts.zshOutput {
		return 2, errors.New("--json and --zsh are mutually exclusive")
	}

	request := strings.TrimSpace(strings.Join(flags.Args(), " "))
	if request == "" {
		usage(stderr)
		return 2, errors.New("missing request")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	s, err := askCodex(ctx, opts, request, stderr)
	if err != nil {
		return 1, err
	}

	switch {
	case opts.jsonOutput:
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(s); err != nil {
			return 1, err
		}
	case opts.zshOutput:
		fmt.Fprintf(stdout, "HOWDO_COMMAND=%s\n", zshQuote(s.Command))
		fmt.Fprintf(stdout, "HOWDO_RATIONALE=%s\n", zshQuote(s.Rationale))
	default:
		fmt.Fprintf(stdout, "Reason: %s\n\n%s\n", s.Rationale, s.Command)
	}

	return 0, nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: howdo [--json|--zsh] [--no-spinner] <request>")
	fmt.Fprintln(w, `Example: howdo show current harddisk space`)
}

func askCodex(ctx context.Context, opts options, request string, stderr io.Writer) (suggestion, error) {
	tmpDir, err := os.MkdirTemp("", "howdo-*")
	if err != nil {
		return suggestion{}, err
	}
	defer os.RemoveAll(tmpDir)

	lastMessagePath := filepath.Join(tmpDir, "last-message.json")
	schemaPath := filepath.Join(tmpDir, "schema.json")
	if err := os.WriteFile(schemaPath, []byte(outputSchema), 0o600); err != nil {
		return suggestion{}, err
	}

	prompt := fmt.Sprintf(codexPrompt, request)
	args := []string{
		"--sandbox", "read-only",
		"-a", "never",
		"exec",
		"--ephemeral",
		"--skip-git-repo-check",
		"--color", "never",
		"--output-schema", schemaPath,
		"--output-last-message", lastMessagePath,
		prompt,
	}

	var codexErr bytes.Buffer
	cmd := exec.CommandContext(ctx, opts.codexPath, args...)
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = &codexErr

	done := make(chan struct{})
	if !opts.noSpinner {
		go spinner(stderr, done)
	}
	err = cmd.Run()
	close(done)
	if !opts.noSpinner {
		fmt.Fprint(stderr, "\r\033[K")
	}
	if err != nil {
		detail := strings.TrimSpace(codexErr.String())
		if detail != "" {
			return suggestion{}, fmt.Errorf("codex failed: %w: %s", err, detail)
		}
		return suggestion{}, fmt.Errorf("codex failed: %w", err)
	}

	raw, err := os.ReadFile(lastMessagePath)
	if err != nil {
		return suggestion{}, fmt.Errorf("read codex output: %w", err)
	}
	s, err := parseSuggestion(raw)
	if err != nil {
		return suggestion{}, err
	}
	return s, nil
}

func spinner(w io.Writer, done <-chan struct{}) {
	frames := []string{"-", "\\", "|", "/"}
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			fmt.Fprintf(w, "\r%s asking codex...", frames[i%len(frames)])
			i++
		}
	}
}

func parseSuggestion(raw []byte) (suggestion, error) {
	var s suggestion
	if err := json.Unmarshal(bytes.TrimSpace(raw), &s); err != nil {
		return suggestion{}, fmt.Errorf("parse codex JSON: %w", err)
	}
	s.Command = strings.TrimSpace(s.Command)
	s.Rationale = strings.TrimSpace(s.Rationale)
	if s.Command == "" {
		return suggestion{}, errors.New("codex returned an empty command")
	}
	if strings.ContainsRune(s.Command, '\x00') || strings.ContainsRune(s.Rationale, '\x00') {
		return suggestion{}, errors.New("codex returned invalid NUL bytes")
	}
	if strings.Contains(s.Command, "\n") || strings.Contains(s.Command, "\r") {
		return suggestion{}, errors.New("codex returned a multi-line command")
	}
	if s.Rationale == "" {
		return suggestion{}, errors.New("codex returned an empty rationale")
	}
	return s, nil
}

func zshQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
