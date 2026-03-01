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
sed -i "s|src=\"https://files.catbox.moe/[^\"]*\" alt=\"upload\"|src=\"${urls[upload]}\" alt=\"upload\"|" README.md
sed -i "s|src=\"https://files.catbox.moe/[^\"]*\" alt=\"gallery\"|src=\"${urls[gallery]}\" alt=\"gallery\"|" README.md
sed -i "s|src=\"https://files.catbox.moe/[^\"]*\" alt=\"modal\"|src=\"${urls[modal]}\" alt=\"modal\"|" README.md
sed -i "s|src=\"https://files.catbox.moe/[^\"]*\" alt=\"admin\"|src=\"${urls[admin]}\" alt=\"admin\"|" README.md

echo "Don't forget to commit the updated README.md!"
