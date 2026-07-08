#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
PACKAGE_NAME="${1:-clipsync-windows-amd64}"
STAGE_DIR="$DIST_DIR/$PACKAGE_NAME"
ZIP_PATH="$DIST_DIR/$PACKAGE_NAME.zip"

rm -rf "$STAGE_DIR" "$ZIP_PATH"
mkdir -p "$STAGE_DIR"

echo "==> Building clipsync.exe (windows/amd64)..."
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$STAGE_DIR/clipsync.exe" "$ROOT_DIR/cmd/clipsync"

echo "==> Copying assets..."
cp "$ROOT_DIR/LICENSE" "$STAGE_DIR/"
cp "$ROOT_DIR/README.md" "$STAGE_DIR/"
cp "$ROOT_DIR/docs/USAGE.md" "$STAGE_DIR/USAGE.md"

echo "==> Packaging..."
if command -v zip >/dev/null 2>&1; then
  (cd "$DIST_DIR" && zip -qr "$ZIP_PATH" "$PACKAGE_NAME")
else
  python3 - "$DIST_DIR" "$PACKAGE_NAME" "$ZIP_PATH" <<'PYEOF'
import os, sys, zipfile
dist_dir, pkg_name, zip_path = sys.argv[1:4]
root = os.path.join(dist_dir, pkg_name)
with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as ar:
    for curr, dirs, files in os.walk(root):
        dirs.sort(); files.sort()
        rel = os.path.relpath(curr, dist_dir)
        if rel != ".":
            ar.write(curr, rel + "/")
        for name in files:
            ar.write(os.path.join(curr, name), os.path.relpath(os.path.join(curr, name), dist_dir))
PYEOF
fi

echo "==> Package ready: $ZIP_PATH"
echo "    Contents:"
unzip -l "$ZIP_PATH" 2>/dev/null || python3 -c "
import zipfile, sys
with zipfile.ZipFile(sys.argv[1]) as z:
    for f in z.namelist():
        print(f'    {f}')
" "$ZIP_PATH"
