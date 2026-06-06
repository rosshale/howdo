# howdo

`howdo` is a small Go CLI that turns a natural-language request into one shell command using `codex exec`.

With a shell integration loaded:

```sh
howdo show current harddisk space
```

prints a short rationale, then leaves the suggested command ready to edit or run. It never executes the generated command automatically.

## Install

```sh
go install .
```

Make sure your Go bin directory is on `PATH`. For this machine that is usually:

```sh
export PATH="$HOME/go/bin:$PATH"
```

Then source the integration for your shell.

```zsh
# ~/.zshrc
source /path/to/howdo/shell/howdo.zsh
```

```bash
# ~/.bashrc
source /path/to/howdo/shell/howdo.bash
```

Restart your shell or run the `source` command directly.

If you keep the binary somewhere else, set `HOWDO_BIN` before sourcing the integration.

## Usage

```sh
howdo list files sorted by size
```

The zsh wrapper calls the installed binary, prints the rationale, and uses `print -z` to put the command on the next prompt.

The bash wrapper supports two workflows:

```bash
howdo list files sorted by size
```

This prints the command and adds it to history. Press Up to edit or run it.

For prompt insertion in bash, type the natural-language request at the prompt and press `Ctrl-x h`. The wrapper asks Codex, prints the rationale, and replaces the current prompt text with the suggested command.

For debugging or editor integrations:

```sh
howdo --json list files sorted by size
howdo --shell list files sorted by size
howdo --no-spinner list files sorted by size
```

## Safety

`howdo` asks Codex for a single JSON response and instructs it not to run tools. Codex CLI does not currently expose a hard tool-disable flag, so the invocation also uses a read-only sandbox with approvals disabled:

```sh
codex --sandbox read-only -a never exec --ephemeral --skip-git-repo-check
```

Generated commands are inserted for review only. They run only if you press Enter.
