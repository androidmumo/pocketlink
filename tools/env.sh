#!/usr/bin/env bash
# Source this file; no global shell configuration is changed.
if ! command -v go >/dev/null 2>&1 && [[ -x "${HOME}/.local/share/pocketlink/go1.26.8/bin/go" ]]; then
    export PATH="${HOME}/.local/share/pocketlink/go1.26.8/bin:${PATH}"
fi
if command -v go >/dev/null 2>&1; then
    export PATH="$(go env GOPATH)/bin:${PATH}"
fi
