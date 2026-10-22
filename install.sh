#!/usr/bin/env sh
# One-line install:
#   curl -fsSL https://raw.githubusercontent.com/amurru/filetools/main/install.sh | sh
set -eu

REPO="amurru/filetools"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os_name() {
	case "$(uname -s)" in
	Linux) echo "linux" ;;
	Darwin) echo "darwin" ;;
	*)
		echo "unsupported OS: $(uname -s)" >&2
		exit 1
		;;
	esac
}

arch_name() {
	case "$(uname -m)" in
	x86_64 | amd64) echo "amd64" ;;
	aarch64 | arm64) echo "arm64" ;;
	*)
		echo "unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
	esac
}

OS="$(os_name)"
ARCH="$(arch_name)"
ASSET="filetools-${OS}-${ARCH}"

if [ "$OS" = "darwin" ]; then
	echo "Prebuilt binaries are not published for macOS yet; build from source:" >&2
	echo "  go install github.com/amurru/filetools@latest" >&2
	exit 1
fi

API="https://api.github.com/repos/${REPO}/releases/latest"
URL="$(curl -fsSL "$API" | grep -o "\"browser_download_url\": *\"[^\"]*${ASSET}\"" | head -n1 | sed 's/.*"browser_download_url": *"//;s/"$//')"

if [ -z "$URL" ]; then
	echo "Could not find a release asset named ${ASSET}" >&2
	exit 1
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
echo "Downloading ${URL}..."
curl -fsSL "$URL" -o "$TMP"
chmod +x "$TMP"

if [ -w "$INSTALL_DIR" ]; then
	mv "$TMP" "${INSTALL_DIR}/filetools"
else
	echo "Installing to ${INSTALL_DIR} requires elevated privileges."
	sudo mv "$TMP" "${INSTALL_DIR}/filetools"
fi

echo "Installed filetools to ${INSTALL_DIR}/filetools"
filetools version
