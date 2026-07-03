#!/bin/sh
# Anyquery installer
#
#   curl -fsSL https://anyquery.dev/install.sh | sh
#
# Downloads the latest anyquery release for your platform, verifies its
# checksum, and drops the binary somewhere on your PATH. No sudo required.
#
# Environment variables:
#   ANYQUERY_INSTALL_DIR   install into this directory (default: auto)
#   ANYQUERY_VERSION       pin a version, e.g. 0.4.5 (default: latest)
#   NO_COLOR              disable colored output and animations
#
# Homepage: https://anyquery.dev  ·  Source: https://github.com/julien040/anyquery

set -eu

REPO="julien040/anyquery"
BINARY="anyquery"

# ---------------------------------------------------------------------------
# Colors (only when stdout is a terminal and NO_COLOR is unset)
# ---------------------------------------------------------------------------
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
	BOLD=$(printf '\033[1m')
	DIM=$(printf '\033[2m')
	RED=$(printf '\033[31m')
	GREEN=$(printf '\033[32m')
	YELLOW=$(printf '\033[33m')
	CYAN=$(printf '\033[36m')
	MAGENTA=$(printf '\033[35m')
	RESET=$(printf '\033[0m')
else
	BOLD='' DIM='' RED='' GREEN='' YELLOW='' CYAN='' MAGENTA='' RESET=''
fi

step() { printf '%s→%s %s\n' "$CYAN" "$RESET" "$1"; }
ok()   { printf '%s✓%s %s\n' "$GREEN" "$RESET" "$1"; }
warn() { printf '%s!%s %s\n' "$YELLOW" "$RESET" "$1" >&2; }
info() { printf '%s  %s%s\n' "$DIM" "$1" "$RESET"; }
err()  { printf '%s✗ %s%s\n' "$RED" "$1" "$RESET" >&2; exit 1; }

banner() {
	if [ ! -t 1 ] || [ -n "${NO_COLOR:-}" ]; then
		printf 'anyquery installer  ·  https://anyquery.dev\n\n'
		return 0
	fi
	printf '\n%s' "$MAGENTA"
	printf '%s\n' \
' █████╗ ███╗   ██╗██╗   ██╗ ██████╗ ██╗   ██╗███████╗██████╗ ██╗   ██╗' \
'██╔══██╗████╗  ██║╚██╗ ██╔╝██╔═══██╗██║   ██║██╔════╝██╔══██╗╚██╗ ██╔╝' \
'███████║██╔██╗ ██║ ╚████╔╝ ██║   ██║██║   ██║█████╗  ██████╔╝ ╚████╔╝ ' \
'██╔══██║██║╚██╗██║  ╚██╔╝  ██║▄▄ ██║██║   ██║██╔══╝  ██╔══██╗  ╚██╔╝  ' \
'██║  ██║██║ ╚████║   ██║   ╚██████╔╝╚██████╔╝███████╗██║  ██║   ██║   ' \
'╚═╝  ╚═╝╚═╝  ╚═══╝   ╚═╝    ╚══▀▀═╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝   ╚═╝   '
	printf '%s' "$RESET"
	printf '%s          Query anything over SQL  ·  https://anyquery.dev%s\n\n' "$DIM" "$RESET"
}

have() { command -v "$1" >/dev/null 2>&1; }

# ---------------------------------------------------------------------------
# Detect platform. Release archives are named to match `uname`, so Darwin/Linux
# map straight through; only the machine arch needs normalizing.
# ---------------------------------------------------------------------------
detect_platform() {
	os=$(uname -s)
	arch=$(uname -m)

	case "$os" in
		Darwin | Linux) ;;
		MINGW* | MSYS* | CYGWIN* | Windows_NT)
			err "Windows isn't supported by this script. Use Scoop or Winget — see https://anyquery.dev/docs/#installation" ;;
		*)
			err "Unsupported operating system: $os" ;;
	esac

	case "$arch" in
		x86_64 | amd64) arch="x86_64" ;;
		arm64 | aarch64) arch="arm64" ;;
		*)
			err "Unsupported architecture: $arch (only x86_64 and arm64 are available)" ;;
	esac

	OS="$os"
	ARCH="$arch"
	ASSET="${BINARY}_${OS}_${ARCH}.tar.gz"
}

# download <url> <output-file>
download() {
	if [ "$DOWNLOADER" = "curl" ]; then
		curl -fsSL "$1" -o "$2"
	else
		wget -q "$1" -O "$2"
	fi
}

# Run a command in the background with a spinner (falls back to a plain line
# when stdout is not a terminal). Returns the command's exit status.
run_with_spinner() {
	label="$1"
	shift
	if [ ! -t 1 ] || [ -n "${NO_COLOR:-}" ]; then
		printf '%s\n' "$label"
		"$@"
		return $?
	fi

	"$@" &
	pid=$!
	spin='|/-\'
	i=0
	while kill -0 "$pid" 2>/dev/null; do
		i=$(( (i + 1) % 4 ))
		c=$(printf '%s' "$spin" | cut -c $((i + 1)))
		printf '\r%s%s%s %s ' "$CYAN" "$c" "$RESET" "$label"
		sleep 0.1
	done
	printf '\r\033[K'
	if ! wait "$pid"; then
		return 1
	fi
	return 0
}

# verify_checksum <file> <checksums-file> <asset-name>
verify_checksum() {
	file="$1"
	sums="$2"
	name="$3"

	expected=$(grep " ${name}\$" "$sums" 2>/dev/null | awk '{print $1}' | head -n1)
	[ -n "$expected" ] || err "No checksum found for ${name} in checksums.txt"

	if have sha256sum; then
		actual=$(sha256sum "$file" | awk '{print $1}')
	elif have shasum; then
		actual=$(shasum -a 256 "$file" | awk '{print $1}')
	else
		warn "No sha256 tool found (sha256sum/shasum); skipping checksum verification."
		return 0
	fi

	[ "$expected" = "$actual" ] ||
		err "Checksum mismatch for ${name}. Expected ${expected}, got ${actual}. Aborting."
}

choose_install_dir() {
	if [ -n "${ANYQUERY_INSTALL_DIR:-}" ]; then
		printf '%s\n' "${ANYQUERY_INSTALL_DIR}"
	elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
		printf '%s\n' "/usr/local/bin"
	else
		printf '%s\n' "${XDG_BIN_HOME:-$HOME/.local/bin}"
	fi
}

on_path() {
	case ":${PATH}:" in
		*":$1:"*) return 0 ;;
		*) return 1 ;;
	esac
}

# Append an `export PATH` line to the right shell profile, once.
ensure_on_path() {
	dir="$1"
	on_path "$dir" && return 0

	case "$(basename "${SHELL:-/bin/sh}")" in
		zsh) profile="$HOME/.zshrc" ;;
		bash) profile="$HOME/.bashrc" ;;
		*) profile="$HOME/.profile" ;;
	esac

	if [ -f "$profile" ] && grep -qF "$dir" "$profile" 2>/dev/null; then
		PATH_NOTE="$profile"
		return 0
	fi

	printf '\n# Added by the anyquery installer\nexport PATH="%s:$PATH"\n' "$dir" >>"$profile" ||
		{ warn "Could not update $profile; add $dir to your PATH manually."; return 0; }
	PATH_ADDED="$profile"
}

pm_hint() {
	if [ "$OS" = "Darwin" ] && have brew; then
		info "Tip: 'brew install anyquery' installs it with managed auto-updates."
	elif [ "$OS" = "Linux" ]; then
		if have apt-get; then
			info "Tip: anyquery is also on APT (apt.julienc.me) for managed updates."
		elif have dnf || have yum; then
			info "Tip: anyquery is also on YUM/DNF (yum.julienc.me) for managed updates."
		fi
	fi
}

main() {
	banner

	# Pick a downloader.
	if have curl; then
		DOWNLOADER="curl"
	elif have wget; then
		DOWNLOADER="wget"
	else
		err "Need either curl or wget installed to download anyquery."
	fi

	detect_platform

	# Resolve the release base URL. CLI release tags are bare (0.4.5), so strip
	# any leading "v". Archive filenames never carry a version.
	version="${ANYQUERY_VERSION:-}"
	version="${version#v}"
	if [ -n "$version" ]; then
		base="https://github.com/${REPO}/releases/download/${version}"
		step "Installing anyquery ${BOLD}${version}${RESET} for ${OS}/${ARCH}"
	else
		base="https://github.com/${REPO}/releases/latest/download"
		step "Installing the latest anyquery for ${OS}/${ARCH}"
	fi

	tmp=$(mktemp -d 2>/dev/null || mktemp -d -t anyquery)
	trap 'rm -rf "$tmp"' EXIT INT TERM

	run_with_spinner "Downloading ${ASSET}..." download "${base}/${ASSET}" "$tmp/$ASSET" ||
		err "Failed to download ${base}/${ASSET}. Is the version correct? See https://github.com/${REPO}/releases"

	download "${base}/checksums.txt" "$tmp/checksums.txt" ||
		err "Failed to download checksums from ${base}/checksums.txt"

	verify_checksum "$tmp/$ASSET" "$tmp/checksums.txt" "$ASSET"
	ok "Checksum verified"

	tar -xzf "$tmp/$ASSET" -C "$tmp" || err "Failed to extract $ASSET"
	[ -f "$tmp/$BINARY" ] || err "The '$BINARY' binary was not found inside the archive"
	chmod +x "$tmp/$BINARY"

	target_dir=$(choose_install_dir)
	mkdir -p "$target_dir" || err "Could not create install directory: $target_dir"
	mv -f "$tmp/$BINARY" "$target_dir/$BINARY" ||
		err "Could not write to $target_dir. Set ANYQUERY_INSTALL_DIR to a writable directory and retry."

	ensure_on_path "$target_dir"

	installed_version=$("$target_dir/$BINARY" --version 2>/dev/null | head -n1 || true)

	printf '\n'
	ok "Installed ${BOLD}anyquery${RESET} → ${BOLD}${target_dir}/${BINARY}${RESET}"
	[ -n "${installed_version:-}" ] && info "$installed_version"

	if [ -n "${PATH_ADDED:-}" ]; then
		printf '\n'
		warn "Added ${target_dir} to your PATH in ${PATH_ADDED}"
		info "Restart your shell, or run:  export PATH=\"${target_dir}:\$PATH\""
	elif [ -n "${PATH_NOTE:-}" ]; then
		info "${target_dir} is already set up in ${PATH_NOTE}."
	fi

	pm_hint

	printf '\n%sGet started%s\n' "$BOLD" "$RESET"
	printf '  %sanyquery%s                    open the interactive shell\n' "$CYAN" "$RESET"
	printf '  %sanyquery --help%s             see all commands\n' "$CYAN" "$RESET"
	printf '  %sanyquery install <plugin>%s   add a data source\n' "$CYAN" "$RESET"
	printf '\n%sTo update later:%s re-run this installer, or use your package manager.\n' "$DIM" "$RESET"
}

main "$@"
