#!/usr/bin/env sh
# Mithril framework installer — installs the global mithril CLI via go install.
set -e

REPO="github.com/mithril-framework/mithril/cmd/mithril@v1.0.2"
EXPECTED_PREFIX="mithril 1."

echo "Installing Mithril CLI..."

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is not installed. Install Go 1.25+ from https://go.dev/dl/" >&2
  exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Found Go $GO_VERSION"

# Bypass stale proxy.golang.org cache (may still serve v0.1.0 as @latest).
GOPROXY=direct go install "$REPO"

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

NEW_VER=$("$INSTALLED" --version 2>/dev/null || echo unknown)

# Always verify the installed binary — not just PATH shadowing.
case "$NEW_VER" in
  mithril\ 1.*github.com/mithril-framework/mithril*)
    ;;
  *)
    echo "" >&2
    echo "Error: installed mithril CLI is not the expected framework build." >&2
    echo "  Got:      $NEW_VER" >&2
    echo "  Expected: mithril 1.x.x (github.com/mithril-framework/mithril)" >&2
    echo "" >&2
    echo "Try:" >&2
    echo "  GOPROXY=direct go install github.com/mithril-framework/mithril/cmd/mithril@main" >&2
    echo "  export PATH=\"$GOBIN:\$PATH\"" >&2
    echo "  mithril --version" >&2
    exit 1
    ;;
esac

CURRENT=$(command -v mithril 2>/dev/null || true)
CURRENT_VER=""
if [ -n "$CURRENT" ]; then
  CURRENT_VER=$("$CURRENT" --version 2>/dev/null || echo unknown)
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " Required: put Go bin first on your PATH"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  export PATH=\"$GOBIN:\$PATH\""
echo ""
echo "Add that line to ~/.zshrc or ~/.bashrc, then open a new terminal."
echo "Verify:  mithril --version"
echo "Expect:  $NEW_VER"
echo ""

if [ -n "$CURRENT" ] && [ "$CURRENT" != "$INSTALLED" ]; then
  case "$CURRENT_VER" in
    mithril\ *github.com/mithril-framework/mithril*) ;;
    *)
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo " Warning: another mithril is first on PATH"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo ""
      echo "  Found:     $CURRENT"
      echo "  Reports:   $CURRENT_VER"
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

# Fail when a default macOS PATH would run a non-framework binary (unless overridden).
if [ "${MITHRIL_INSTALL_FORCE:-}" != "1" ]; then
  DEFAULT_PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:$GOBIN"
  DEFAULT_BIN=$(PATH="$DEFAULT_PATH" command -v mithril 2>/dev/null || true)
  if [ -n "$DEFAULT_BIN" ]; then
    DEFAULT_VER=$(PATH="$DEFAULT_PATH" "$DEFAULT_BIN" --version 2>/dev/null || echo unknown)
    case "$DEFAULT_VER" in
      mithril\ 1.*github.com/mithril-framework/mithril*) ;;
      *)
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo " Error: wrong mithril would run in a new terminal"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        echo "  Default PATH resolves to: $DEFAULT_BIN"
        echo "  Reports:                  $DEFAULT_VER"
        echo "  Installed framework CLI:    $INSTALLED ($NEW_VER)"
        echo ""
        echo "Fix (recommended — one time):"
        echo "  sudo $INSTALLED init"
        echo ""
        echo "Or add to ~/.zshrc / ~/.bashrc:"
        echo "  export PATH=\"$GOBIN:\$PATH\""
        echo ""
        echo "To skip this check (not recommended): MITHRIL_INSTALL_FORCE=1 sh install.sh"
        exit 1
        ;;
    esac
  fi
fi

echo "Create a project:  mithril new hello-mithril"
echo "Docs:              https://mithril-docs-nine.vercel.app/docs/getting-started/installation"
