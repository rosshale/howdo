# Source this file from ~/.zshrc after installing the howdo binary.
#
#   source /path/to/howdo/shell/howdo.zsh
#
# Usage:
#
#   howdo show current harddisk space
#
# The generated command is placed on the next prompt with zsh's print -z.

howdo() {
  local howdo_bin result exit_code
  howdo_bin="${HOWDO_BIN:-}"
  if [[ -z "$howdo_bin" ]]; then
    howdo_bin="$(whence -p howdo)"
  fi
  if [[ -z "$howdo_bin" && -x "$HOME/go/bin/howdo" ]]; then
    howdo_bin="$HOME/go/bin/howdo"
  fi
  if [[ -z "$howdo_bin" ]]; then
    print -u2 -- "howdo: binary not found; run 'go install .' and ensure it is on PATH"
    return 127
  fi

  result="$("$howdo_bin" --zsh "$@")"
  exit_code=$?
  if (( exit_code != 0 )); then
    return "$exit_code"
  fi

  local HOWDO_COMMAND HOWDO_RATIONALE
  eval "$result"

  if [[ -n "$HOWDO_RATIONALE" ]]; then
    print -r -- "Reason: $HOWDO_RATIONALE"
  fi

  print -z -- "$HOWDO_COMMAND"
}
