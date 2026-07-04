#!/usr/bin/env sh
# Mithril framework installer — installs the global mithril CLI via go install.
set -e

REPO="github.com/mithril-framework/mithril/cmd/mithril@latest"

echo "Installing Mithril CLI..."

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is not installed. Install Go 1.25+ from https://go.dev/dl/" >&2
  exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Found Go $GO_VERSION"

go install "$REPO"

GOBIN=$(go env GOBIN)
if [ -z "$GOBIN" ]; then
  GOBIN="$(go env GOPATH)/bin"
fi

if [ -x "$GOBIN/mithril" ]; then
  echo "Installed: $GOBIN/mithril"
else
  echo "Warning: binary not found at $GOBIN/mithril — ensure $GOBIN is in your PATH" >&2
fi

echo ""
echo "Verify:  mithril --version"
echo "Create:  mithril new hello-mithril"
echo "Docs:    https://mithril-docs-nine.vercel.app/docs/getting-started/installation"
