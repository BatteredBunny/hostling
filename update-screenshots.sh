#!/usr/bin/env bash
set -euo pipefail

echo "Building screenshots..."
out=$(nix build .#example-screenshots --print-out-paths)

declare -A urls
for img in upload gallery modal admin; do
    echo "Uploading ${img}.png to catbox.moe..."
    url=$(curl -F "reqtype=fileupload" -F "fileToUpload=@${out}/screenshots/${img}.png" https://catbox.moe/user/api.php)
    urls[$img]=$url
    echo "  -> $url"
done

echo "Updating README.md..."
sed -i "s|alt=\"upload\" src=\"[^\"]*\"|alt=\"upload\" src=\"${urls[upload]}\"|" README.md
sed -i "s|alt=\"gallery\" src=\"[^\"]*\"|alt=\"gallery\" src=\"${urls[gallery]}\"|" README.md
sed -i "s|alt=\"modal\" src=\"[^\"]*\"|alt=\"modal\" src=\"${urls[modal]}\"|" README.md
sed -i "s|alt=\"admin\" src=\"[^\"]*\"|alt=\"admin\" src=\"${urls[admin]}\"|" README.md

echo "Don't forget to commit the updated README.md!"
