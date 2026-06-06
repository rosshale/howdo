# Source this file from ~/.bashrc after installing the howdo binary.
#
#   source /path/to/howdo/shell/howdo.bash
#
# Usage:
#
#   howdo show current harddisk space
#
# The generated command is printed and added to history. Press Up to edit or run it.
#
# Readline usage:
#
#   Type a request at an empty prompt, then press Ctrl-x h.
#
# The generated command replaces the current prompt text so it can be edited or run.

__howdo_find_bin() {
  local howdo_bin
  howdo_bin="${HOWDO_BIN:-}"
  if [[ -z "$howdo_bin" ]]; then
    howdo_bin="$(type -P howdo 2>/dev/null)"
  fi
  if [[ -z "$howdo_bin" && -x "$HOME/go/bin/howdo" ]]; then
    howdo_bin="$HOME/go/bin/howdo"
  fi
  if [[ -z "$howdo_bin" ]]; then
    printf '%s\n' "howdo: binary not found; run 'go install .' and ensure it is on PATH" >&2
    return 127
  fi
  printf '%s\n' "$howdo_bin"
}

howdo() {
  local howdo_bin result exit_code HOWDO_COMMAND HOWDO_RATIONALE
  howdo_bin="$(__howdo_find_bin)" || return "$?"

  result="$("$howdo_bin" --shell "$@")"
  exit_code=$?
  if (( exit_code != 0 )); then
    return "$exit_code"
  fi

  eval "$result"

  if [[ -n "$HOWDO_RATIONALE" ]]; then
    printf 'Reason: %s\n' "$HOWDO_RATIONALE"
  fi

  printf '%s\n' "$HOWDO_COMMAND"
  history -s "$HOWDO_COMMAND"
  printf '%s\n' "Added to history; press Up to edit or run it."
}

__howdo_readline() {
  local request howdo_bin result exit_code HOWDO_COMMAND HOWDO_RATIONALE
  request="${READLINE_LINE:-}"
  if [[ -z "${request//[[:space:]]/}" ]]; then
    printf '\nhowdo: enter a request before pressing Ctrl-x h\n' >&2
    return 1
  fi

  howdo_bin="$(__howdo_find_bin)" || return "$?"

  printf '\n' >&2
  result="$("$howdo_bin" --shell "$request")"
  exit_code=$?
  if (( exit_code != 0 )); then
    return "$exit_code"
  fi

  eval "$result"

  if [[ -n "$HOWDO_RATIONALE" ]]; then
    printf 'Reason: %s\n' "$HOWDO_RATIONALE" >&2
  fi

  READLINE_LINE="$HOWDO_COMMAND"
  READLINE_POINT=${#READLINE_LINE}
}

if [[ $- == *i* ]]; then
  bind -x '"\C-xh": __howdo_readline'
fi
