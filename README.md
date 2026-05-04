# howdo

`howdo` is a small Go CLI that turns a natural-language request into one shell command using `codex exec`.

With the zsh integration loaded:

```zsh
howdo show current harddisk space
```

prints a short rationale, then places the suggested command on the next prompt ready to edit or run. It never executes the generated command automatically.

## Install

```zsh
go install .
```

Make sure your Go bin directory is on `PATH`. For this machine that is usually:

```zsh
export PATH="$HOME/go/bin:$PATH"
```

Then source the zsh integration from your `.zshrc`:

```zsh
source /Users/rosshale/workspace/howdo/shell/howdo.zsh
```

Restart your shell or run the `source` command directly.

If you keep the binary somewhere else, set `HOWDO_BIN` before sourcing the integration.

## Usage

```zsh
howdo list files sorted by size
```

The zsh wrapper calls the installed binary, prints the rationale, and uses `print -z` to put the command on the next prompt.

For non-zsh use or debugging:

```zsh
howdo --json list files sorted by size
howdo --no-spinner list files sorted by size
```

## Safety

`howdo` asks Codex for a single JSON response and instructs it not to run tools. Codex CLI does not currently expose a hard tool-disable flag, so the invocation also uses a read-only sandbox with approvals disabled:

```zsh
codex --sandbox read-only -a never exec --ephemeral --skip-git-repo-check
```

Generated commands are inserted for review only. They run only if you press Enter.
