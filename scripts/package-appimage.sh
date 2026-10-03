#!/usr/bin/env bash
# Packages the Flutter Linux release bundle as an AppImage.
# Run `flutter build linux --release` in src/flutter first.
# Needs appimagetool on PATH (or APPIMAGETOOL pointing at it).
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
flutter_dir="$root/src/flutter"
bundle="$flutter_dir/build/linux/x64/release/bundle"
appdir="$flutter_dir/build/linux/Roomies.AppDir"
out="${1:-$flutter_dir/build/linux/Roomies-x86_64.AppImage}"
appimagetool="${APPIMAGETOOL:-appimagetool}"

test -x "$bundle/roomies" || { echo "missing $bundle/roomies; run flutter build linux --release" >&2; exit 1; }

rm -rf "$appdir"
mkdir -p "$appdir"
cp -a "$bundle/." "$appdir/"
cp "$flutter_dir/web/icons/Icon-512.png" "$appdir/roomies.png"
cat > "$appdir/roomies.desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=Roomies
Exec=roomies
Icon=roomies
Categories=Office;
DESKTOP
cat > "$appdir/AppRun" <<'APPRUN'
#!/bin/sh
here="$(dirname "$(readlink -f "$0")")"
exec "$here/roomies" "$@"
APPRUN
chmod +x "$appdir/AppRun"

ARCH=x86_64 "$appimagetool" "$appdir" "$out"
echo "$out"
