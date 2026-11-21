#!/bin/sh
set -eu

REPO="oversonv/hexfetch"
API="https://api.github.com/repos/$REPO/releases"
DL="https://github.com/$REPO/releases/download"

say() { printf '%s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

if command -v curl >/dev/null 2>&1; then
    fetch() { curl -fsSL "$1"; }
    fetch_to() { curl -fSL --progress-bar -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
    fetch() { wget -qO- "$1"; }
    fetch_to() { wget -q --show-progress -O "$2" "$1"; }
else
    die "curl or wget is required (Termux: pkg install curl)"
fi

detect_target() {
    os=$(uname -s)
    [ "$os" = "Linux" ] || die "unsupported OS: $os (hexfetch supports Linux and Android/Termux only)"

    case "$(uname -m)" in
        x86_64|amd64)            arch=amd64 ;;
        aarch64|arm64)           arch=arm64 ;;
        armv7l|armv8l|armv7*)    arch=armv7 ;;
        i386|i486|i586|i686)     arch=386 ;;
        riscv64)                 arch=riscv64 ;;
        *) die "unsupported architecture: $(uname -m)" ;;
    esac

    if [ "$(uname -o 2>/dev/null || true)" = "Android" ] || [ -n "${TERMUX_VERSION:-}" ]; then
        [ "$arch" = "arm64" ] || die "Android is supported on arm64 only (found: $arch)"
        echo "android-arm64"
        return
    fi
    echo "linux-$arch"
}

detect_dir() {
    if [ -n "${HEXFETCH_INSTALL_DIR:-}" ]; then
        echo "$HEXFETCH_INSTALL_DIR"
    elif [ -n "${PREFIX:-}" ] && [ -d "$PREFIX/bin" ] && [ "${PREFIX#*com.termux}" != "$PREFIX" ]; then
        echo "$PREFIX/bin"
    elif [ "$(id -u)" = "0" ]; then
        echo "/usr/local/bin"
    else
        echo "$HOME/.local/bin"
    fi
}

latest_tag() {
    fetch "$API/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

sha_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | cut -d' ' -f1
    else
        return 1
    fi
}

main() {
    target=$(detect_target)
    dir=$(detect_dir)

    if [ -n "${HEXFETCH_VERSION:-}" ]; then
        version=${HEXFETCH_VERSION#v}
        tag="v$version"
    else
        say "Looking up the latest release..."
        tag=$(latest_tag)
        [ -n "$tag" ] || die "could not determine the latest release (GitHub API unreachable or rate-limited)"
        version=${tag#v}
    fi

    file="hexfetch_${version}_${target}"
    say "Platform: $target"
    say "Version:  $version"
    say "Target:   $dir/hexfetch"

    tmp=$(mktemp -d 2>/dev/null || mktemp -d -t hexfetch)
    trap 'rm -rf "$tmp"' EXIT INT TERM

    say "Downloading $file..."
    fetch_to "$DL/$tag/$file" "$tmp/$file" || die "download failed: $DL/$tag/$file"

    if fetch "$DL/$tag/SHA256SUMS" > "$tmp/SHA256SUMS" 2>/dev/null; then
        want=$(grep "  $file\$" "$tmp/SHA256SUMS" | cut -d' ' -f1 || true)
        if [ -z "$want" ]; then
            die "$file is not listed in SHA256SUMS"
        fi
        if got=$(sha_of "$tmp/$file"); then
            [ "$got" = "$want" ] || die "checksum mismatch (expected $want, got $got)"
            say "Checksum OK"
        else
            say "warning: no sha256sum/shasum found, checksum not verified"
        fi
    else
        say "warning: SHA256SUMS not available, checksum not verified"
    fi

    chmod +x "$tmp/$file"
    mkdir -p "$dir" 2>/dev/null || die "cannot create $dir (try sudo, or set HEXFETCH_INSTALL_DIR)"
    [ -w "$dir" ] || die "$dir is not writable (try sudo, or set HEXFETCH_INSTALL_DIR)"
    mv -f "$tmp/$file" "$dir/hexfetch"

    say "Installed: $dir/hexfetch"

    case ":$PATH:" in
        *":$dir:"*) ;;
        *)
            say ""
            say "$dir is not in your PATH. Add it:"
            case "${SHELL:-}" in
                */zsh)  rc="~/.zshrc" ;;
                */bash) rc="~/.bashrc" ;;
                *)      rc="~/.profile" ;;
            esac
            say "  echo 'export PATH=\"$dir:\$PATH\"' >> $rc"
            say "  then restart the terminal."
            ;;
    esac

    say ""
    say "Run:  hexfetch    (config: hexfetch --gen-config)"
}

main "$@"

