#!/bin/sh
# Installs the latest via-terminal release on Linux or macOS:
#   curl -fsSL https://raw.githubusercontent.com/bulkinglb/via-terminal/master/install.sh | sh
# VERSION=v0.1.0 picks another release, INSTALL_DIR another folder.
set -eu

repo=bulkinglb/via-terminal
dir=${INSTALL_DIR:-/usr/local/bin}

case $(uname -s) in
Linux) os=linux ;;
Darwin) os=darwin ;;
*)
	echo "no installer for $(uname -s); on Windows, download the .exe from https://github.com/$repo/releases" >&2
	exit 1
	;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*)
	echo "no release for $(uname -m); try: go install github.com/$repo@latest" >&2
	exit 1
	;;
esac

if [ -n "${VERSION:-}" ]; then
	url=https://github.com/$repo/releases/download/$VERSION
else
	url=https://github.com/$repo/releases/latest/download
fi
name=via-terminal-$os-$arch

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "downloading $name.tar.gz"
curl -fsSL -o "$tmp/$name.tar.gz" "$url/$name.tar.gz"
curl -fsSL -o "$tmp/checksums.txt" "$url/checksums.txt"

if command -v sha256sum >/dev/null; then
	sum="sha256sum"
else
	sum="shasum -a 256"
fi
if ! (cd "$tmp" && grep " $name.tar.gz\$" checksums.txt | $sum -c -) >/dev/null 2>&1; then
	echo "the download doesn't match the release's checksum, not installing" >&2
	exit 1
fi
tar -xzf "$tmp/$name.tar.gz" -C "$tmp"

# sudo only when the folder isn't writable, e.g. /usr/local/bin.
sudo=
if ! mkdir -p "$dir" 2>/dev/null || [ ! -w "$dir" ]; then
	sudo=sudo
fi
$sudo mkdir -p "$dir"
$sudo install -m 755 "$tmp/$name/via-terminal" "$dir/via-terminal"
echo "installed $("$dir/via-terminal" -version) to $dir"

case ":$PATH:" in
*":$dir:"*) ;;
*) echo "note: $dir isn't on your PATH yet" ;;
esac
if [ "$os" = linux ] && [ ! -e /etc/udev/rules.d/99-via.rules ]; then
	echo "if via-terminal can't open your keyboard, allow access once and replug it:"
	echo "  echo 'KERNEL==\"hidraw*\", SUBSYSTEM==\"hidraw\", MODE=\"0660\", TAG+=\"uaccess\"' | sudo tee /etc/udev/rules.d/99-via.rules"
fi
