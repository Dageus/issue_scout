#!/usr/bin/env bash
# PostToolUse hook: after Claude writes/edits a .go file, run gofmt on it
# and go vet on its package. Exit 2 sends stderr back to Claude so it
# fixes what it broke; exit 0 means nothing to report.
set -u

f=$(jq -r '.tool_input.file_path // empty')
[[ "$f" == *.go && -f "$f" ]] || exit 0

if ! command -v go >/dev/null; then
  echo "go-check: go not on PATH (start claude inside the direnv shell)" >&2
  exit 0
fi

root=${CLAUDE_PROJECT_DIR:-$(pwd)}
rel=$(realpath --relative-to="$root" "$f")
[[ "$rel" == ../* ]] && exit 0 # outside the project

cd "$root" || exit 0

if ! out=$(gofmt -w "$rel" 2>&1); then
  echo "gofmt failed on $rel:" >&2
  echo "$out" >&2
  exit 2
fi

if ! out=$(go vet "./$(dirname "$rel")" 2>&1); then
  echo "go vet failed for ./$(dirname "$rel"):" >&2
  echo "$out" >&2
  exit 2
fi
