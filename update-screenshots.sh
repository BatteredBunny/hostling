#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

declare -A shots=(
    [upload]=upload/default
    [gallery]=gallery/default
    [modal]=preview/image
    [admin]=admin/default
)

echo "Building screenshots..."
out=$(nix build .#screenshot --no-link --print-out-paths)

for alt in "${!shots[@]}"; do
    file="$out/screenshots/${shots[$alt]}.png"
    [[ -f $file ]] || { echo "Missing capture: $file" >&2; exit 1; }

    echo "Uploading ${shots[$alt]}.png..."
    url=$(curl --fail-with-body --silent --show-error --retry 3 \
        -F reqtype=fileupload -F "fileToUpload=@$file" \
        https://catbox.moe/user/api.php)
    [[ $url =~ ^https://files\.catbox\.moe/[A-Za-z0-9]+\.png$ ]] ||
        { echo "Unexpected catbox response: $url" >&2; exit 1; }
    echo "  -> $url"

    sed -i "s|alt=\"$alt\" src=\"[^\"]*\"|alt=\"$alt\" src=\"$url\"|" README.md
done

echo "Don't forget to commit the updated README.md!"
