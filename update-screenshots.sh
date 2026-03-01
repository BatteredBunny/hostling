#!/usr/bin/env bash
set -euo pipefail

echo "Building screenshots..."
out=$(nix build .#example-screenshots --print-out-paths)

mkdir -p screenshots
cp "$out/screenshots/upload.png" screenshots/upload.png
cp "$out/screenshots/gallery.png" screenshots/gallery.png
cp "$out/screenshots/modal.png" screenshots/modal.png
cp "$out/screenshots/admin.png" screenshots/admin.png

echo "Updated screenshots/"
ls -lh screenshots/
