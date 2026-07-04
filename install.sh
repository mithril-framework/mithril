#!/usr/bin/env sh
# Mithril framework installer — installs the global mithril CLI via go install.
set -e

REPO="github.com/mithril-framework/mithril/cmd/mithril@main"

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

# Detect an older scaffold CLI shadowing the real binary on PATH.
if command -v mithril >/dev/null 2>&1; then
  VER=$(PATH="$GOBIN:$PATH" mithril --version 2>/dev/null || true)
  case "$VER" in
    mithril\ *github.com/mithril-framework/mithril*)
      ;;
    mithril\ 1.*|mithril\ 0.*)
      if ! echo "$VER" | grep -q "github.com/mithril-framework/mithril"; then
        echo ""
        echo "Warning: 'mithril' on PATH may not be this installer build:"
        echo "  $VER"
        echo "Ensure $(go env GOPATH)/bin is before /usr/local/bin in PATH, or run:"
        echo "  sudo $GOBIN/mithril init"
      fi
      ;;
    dev|*scaffold*)
      echo ""
      echo "Warning: another 'mithril' CLI is on PATH ($VER)."
      echo "Run: sudo $GOBIN/mithril init"
      echo "Or prepend PATH: export PATH=\"$GOBIN:\$PATH\""
      ;;
  esac
fi

echo ""
echo "Verify:  mithril --version"
echo "Create:  mithril new hello-mithril"
echo "Docs:    https://mithril-docs-nine.vercel.app/docs/getting-started/installation"
