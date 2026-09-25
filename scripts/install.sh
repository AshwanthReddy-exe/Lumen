#!/bin/sh
# Lumen installer.
#
#   curl -fsSL https://github.com/AshwanthReddy-exe/Lumen/releases/latest/download/install.sh | sh
#
# Detects the host platform, resolves the matching artifact from the pinned
# release manifest, verifies its SHA-256 before writing anything, and installs
# it. The installer never runs `lumen-host init` and never touches Host state;
# initializing a Space stays an explicit, separate step.
#
# Environment:
#   LUMEN_MANIFEST       manifest path or https URL (default: the latest release)
#   LUMEN_ARTIFACT_DIR   read artifacts from this directory instead of the network
#   LUMEN_INSTALL_DIR    install directory (default: ~/.local/bin, or /usr/local/bin)
#   LUMEN_VERSION        require a specific version, for example 0.1.0-m1
set -eu

install_dir=${LUMEN_INSTALL_DIR:-}
version_required=${LUMEN_VERSION:-}
manifest_source=${LUMEN_MANIFEST:-https://github.com/AshwanthReddy-exe/Lumen/releases/latest/download/manifest-v1.json}
artifact_dir=${LUMEN_ARTIFACT_DIR:-}
dry_run=0

while [ "$#" -gt 0 ]; do
	case "$1" in
		--dry-run) dry_run=1; shift ;;
		--install-dir) install_dir=$2; shift 2 ;;
		-h|--help)
			sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
			exit 0 ;;
		*) printf 'install.sh: unknown argument %s\n' "$1" >&2; exit 1 ;;
	esac
done

fail() {
	printf 'install.sh: %s\n' "$1" >&2
	exit 1
}

# Resolve the host platform using the same names the release manifest uses.
detect_target() {
	os=$(uname -s 2>/dev/null || printf unknown)
	case "$os" in
		Linux)
			# Termux is an Android userland and reports Linux.
			if [ -n "${PREFIX:-}" ] && [ -d "${PREFIX:-}/bin" ] && [ -n "${ANDROID_ROOT:-}" ]; then
				printf 'android'
			else
				printf 'linux'
			fi ;;
		Darwin) printf 'darwin' ;;
		*) fail "unsupported operating system $os" ;;
	esac
}

target_os=$(detect_target)
machine=$(uname -m 2>/dev/null || printf unknown)
case "$machine" in
	x86_64|amd64) target_arch=amd64 ;;
	aarch64|arm64) target_arch=arm64 ;;
	*) fail "unsupported architecture $machine" ;;
esac
if [ "$target_os" = android ] && [ "$target_arch" != arm64 ]; then
	fail 'Termux is supported on arm64 only'
fi

work_dir=${TMPDIR:-/tmp}/lumen-install.$$
mkdir -m 700 -p "$work_dir"
cleanup() { rm -rf -- "$work_dir"; }
trap cleanup EXIT

if [ -n "$artifact_dir" ]; then
	[ -d "$artifact_dir" ] || fail "artifact directory $artifact_dir does not exist"
	manifest_file=$work_dir/manifest.json
	if [ -f "$manifest_source" ]; then
		cp -- "$manifest_source" "$manifest_file"
	else
		fail 'LUMEN_ARTIFACT_DIR requires a local manifest path'
	fi
else
	command -v curl >/dev/null 2>&1 || fail 'curl is required'
	manifest_file=$work_dir/manifest.json
	case "$manifest_source" in
		https://*) curl -fsSL --retry 3 --connect-timeout 10 --max-time 120 -o "$manifest_file" "$manifest_source" ;;
		*) fail 'manifest source must be an https URL or a local file' ;;
	esac
fi

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	else
		shasum -a 256 "$1" | cut -d' ' -f1
	fi
}

# The manifest is a compact JSON document. Select the single artifact object for
# this host and pull its fields out without requiring jq.
artifact_object=$(tr '{' '\n' <"$manifest_file" |
	grep "\"name\":\"lumen\"" |
	grep "\"os\":\"$target_os\"" |
	grep "\"architecture\":\"$target_arch\"" |
	head -n 1)
[ -n "$artifact_object" ] || fail "manifest has no lumen artifact for $target_os/$target_arch"

field() {
	printf '%s' "$artifact_object" | sed -n "s/.*\"$1\":\"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p"
}
asset_version=$(field version)
asset_url=$(field url)
asset_size=$(field size)
asset_digest=$(field sha256)
asset_mode=$(field executableMode)

[ -n "$asset_url" ] || fail 'manifest artifact has no download URL'
[ -n "$asset_digest" ] || fail 'manifest artifact has no digest'
case "$asset_digest" in
	sha256:*) ;;
	*) fail 'manifest artifact digest is not a sha256 digest' ;;
esac
printf '%s' "$asset_digest" | grep -Eq '^sha256:[0-9a-f]{64}$' || fail 'manifest artifact digest is malformed'
[ "$asset_digest" != "sha256:0000000000000000000000000000000000000000000000000000000000000000" ] || fail 'manifest artifact digest is a placeholder'

if [ -n "$version_required" ] && [ "$asset_version" != "$version_required" ]; then
	fail "manifest offers $asset_version but $version_required was requested"
fi

if [ -z "$install_dir" ]; then
	if [ -w /usr/local/bin ] 2>/dev/null; then
		install_dir=/usr/local/bin
	else
		install_dir=${HOME:?HOME is required}/.local/bin
	fi
fi

printf 'install.sh: host %s/%s\n' "$target_os" "$target_arch"
printf 'install.sh: version %s\n' "$asset_version"
printf 'install.sh: install %s\n' "$install_dir/lumen-host"

staged=$work_dir/lumen-host
if [ -n "$artifact_dir" ]; then
	asset_name=${asset_url##*/}
	cp -- "$artifact_dir/$asset_name" "$staged" || fail "artifact $asset_name is missing from $artifact_dir"
else
	curl -fsSL --retry 3 --connect-timeout 10 --max-time 300 -o "$staged" "$asset_url" || fail 'artifact download failed'
fi

actual_size=$(wc -c <"$staged" | tr -d ' ')
[ "$actual_size" = "$asset_size" ] || fail "artifact size $actual_size does not match the manifest size $asset_size"
actual_digest="sha256:$(sha256_of "$staged")"
[ "$actual_digest" = "$asset_digest" ] || fail 'artifact digest does not match the manifest'

if [ "$dry_run" -eq 1 ]; then
	printf 'install.sh: verified %s (%s bytes); dry run, nothing installed\n' "$actual_digest" "$actual_size"
	exit 0
fi

mkdir -p -- "$install_dir" || fail "cannot create $install_dir"
chmod "$asset_mode" "$staged"
mv -f -- "$staged" "$install_dir/lumen-host"
chmod "$asset_mode" "$install_dir/lumen-host"

printf 'install.sh: installed %s\n' "$install_dir/lumen-host"
printf '%s\n' 'Next: set LUMEN_DATA_DIR to a private directory, then run `lumen setup`.'
