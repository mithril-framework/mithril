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

INSTALLED="$GOBIN/mithril"
if [ ! -x "$INSTALLED" ]; then
  echo "Error: binary not found at $INSTALLED" >&2
  exit 1
fi
echo "Installed: $INSTALLED"

# What does the user's current PATH resolve?
CURRENT=$(command -v mithril 2>/dev/null || true)
CURRENT_VER=""
if [ -n "$CURRENT" ]; then
  CURRENT_VER=$("$CURRENT" --version 2>/dev/null || echo unknown)
fi
NEW_VER=$("$INSTALLED" --version 2>/dev/null || echo unknown)

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " Required: put Go bin first on your PATH"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  export PATH=\"$GOBIN:\$PATH\""
echo ""
echo "Add that line to ~/.zshrc or ~/.bashrc, then open a new terminal."
echo "Verify:  mithril --version"
echo "Expect:  mithril 1.0.0 (github.com/mithril-framework/mithril)"
echo ""

if [ -n "$CURRENT" ] && [ "$CURRENT" != "$INSTALLED" ]; then
  case "$CURRENT_VER" in
    mithril\ *github.com/mithril-framework/mithril*) ;;
    *)
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo " Warning: another mithril is first on PATH"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo ""
      echo "  Found:    $CURRENT"
      echo "  Reports:  $CURRENT_VER"
      echo "  Installed: $INSTALLED ($NEW_VER)"
      echo ""
      echo "Fix for this shell:"
      echo "  export PATH=\"$GOBIN:\$PATH\""
      echo ""
      echo "Fix system-wide (replaces /usr/local/bin/mithril):"
      echo "  sudo $INSTALLED init"
      echo ""
      ;;
  esac
fi

echo "Create a project:  mithril new hello-mithril"
echo "Docs:              https://mithril-docs-nine.vercel.app/docs/getting-started/installation"
