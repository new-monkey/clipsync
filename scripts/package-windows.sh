#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
PACKAGE_NAME="${1:-clipsync-windows-amd64}"
STAGE_DIR="$DIST_DIR/$PACKAGE_NAME"
ZIP_PATH="$DIST_DIR/$PACKAGE_NAME.zip"

export GOOS=windows
export GOARCH=amd64

mkdir -p "$DIST_DIR"
rm -rf "$STAGE_DIR" "$ZIP_PATH"

go build -trimpath -ldflags "-s -w" -o "$DIST_DIR/clipsync-server.exe" "$ROOT_DIR/cmd/server"
go build -trimpath -ldflags "-s -w" -o "$DIST_DIR/clipsync-client.exe" "$ROOT_DIR/cmd/client"

mkdir -p "$STAGE_DIR/configs" "$STAGE_DIR/scripts"

cp "$DIST_DIR/clipsync-server.exe" "$STAGE_DIR/"
cp "$DIST_DIR/clipsync-client.exe" "$STAGE_DIR/"
cp "$ROOT_DIR/configs/server.json" "$STAGE_DIR/configs/"
cp "$ROOT_DIR/configs/client.json" "$STAGE_DIR/configs/"
cp "$ROOT_DIR/README.md" "$STAGE_DIR/"
cp "$ROOT_DIR/LICENSE" "$STAGE_DIR/" 2>/dev/null || true
cp "$ROOT_DIR/scripts/install-autostart-client.ps1" "$STAGE_DIR/scripts/"
cp "$ROOT_DIR/scripts/install-autostart-server.ps1" "$STAGE_DIR/scripts/"
cp "$ROOT_DIR/scripts/uninstall-autostart.ps1" "$STAGE_DIR/scripts/"

if command -v zip >/dev/null 2>&1; then
  (
    cd "$DIST_DIR"
    zip -qr "$ZIP_PATH" "$PACKAGE_NAME"
  )
else
  python3 - "$DIST_DIR" "$PACKAGE_NAME" "$ZIP_PATH" <<'PY'
import os
import sys
import zipfile

dist_dir, package_name, zip_path = sys.argv[1:4]
root = os.path.join(dist_dir, package_name)

with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
    for current_root, dirs, files in os.walk(root):
        dirs.sort()
        files.sort()
        rel_root = os.path.relpath(current_root, dist_dir)
        if rel_root != ".":
            archive.write(current_root, rel_root + "/")
        for name in files:
            full_path = os.path.join(current_root, name)
            rel_path = os.path.relpath(full_path, dist_dir)
            archive.write(full_path, rel_path)
PY
fi

echo "Package ready: $ZIP_PATH"
echo "Unpacked root: $STAGE_DIR"