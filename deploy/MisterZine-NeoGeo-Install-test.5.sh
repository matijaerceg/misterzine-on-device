#!/bin/bash
# Exact-version trial installer. Does not register an update feed.
set -eu
CARD=/media/fat
URL=https://github.com/matijaerceg/misterzine-on-device/releases/download/v1.2.24-neogeo-test.5/MisterZine-NeoGeo-test.5.zip
SHA=a78c7ee6f62e892cef06cb57b5fd1c5cd55d0c7ba1d78439f768ebc727002ad4
ROOT="$CARD/misterzine-neogeo-test"
ENTRY=MisterZine-NeoGeo-TEST.sh
echo "Installing MisterZine Neo Geo test 5"
if pidof mz-neogeo >/dev/null 2>&1; then
  echo "Exit the Neo Geo test app and its running game before installing."
  exit 1
fi
for tool in curl sha256sum unzip mktemp; do
  command -v "$tool" >/dev/null || { echo "Missing required tool: $tool"; exit 1; }
done
[ -d "$CARD/Scripts" ] || { echo "MiSTer Scripts folder not found."; exit 1; }
[ ! -L "$ROOT" ] && [ ! -L "$CARD/Scripts/$ENTRY" ] || { echo "Refusing a linked test installation."; exit 1; }
WORK=$(mktemp -d "$CARD/.mz-neogeo-install.XXXXXX")
trap 'rm -rf -- "$WORK"' EXIT
echo "Downloading the fixed test release..."
# Downloader maintains a newer trust store than many MiSTer Linux images.
CA_ARGS=()
if [ -s "$CARD/Scripts/.config/downloader/cacert.pem" ]; then
  CA_ARGS=(--cacert "$CARD/Scripts/.config/downloader/cacert.pem")
fi
if ! curl "${CA_ARGS[@]}" --fail --location --retry 3 --connect-timeout 20 --max-time 300 --proto '=https' --proto-redir '=https' "$URL" -o "$WORK/test.zip"; then
  echo "Download failed. For certificate errors, run Update All to refresh its certificates, or extract the test ZIP manually."
  exit 1
fi
echo "$SHA  $WORK/test.zip" | sha256sum -c -
unzip -tq "$WORK/test.zip"
mkdir "$WORK/staged"
if [ -d "$ROOT" ]; then
  cp -a "$ROOT" "$WORK/staged/misterzine-neogeo-test"
fi
unzip -oq "$WORK/test.zip" -d "$WORK/staged"
PAYLOAD="$WORK/staged/misterzine-neogeo-test"
chmod +x "$PAYLOAD/mz-neogeo" "$PAYLOAD/mzmglprobe" "$PAYLOAD/launch.sh" "$WORK/staged/Scripts/$ENTRY"
"$PAYLOAD/mz-neogeo" -version
BACKUP="$CARD/misterzine-neogeo-test-backup-$(basename "$WORK")"
if [ -d "$ROOT" ]; then mv "$ROOT" "$BACKUP"; fi
if ! mv "$PAYLOAD" "$ROOT"; then
  [ ! -d "$BACKUP" ] || mv "$BACKUP" "$ROOT"
  exit 1
fi
if ! cp "$WORK/staged/Scripts/$ENTRY" "$CARD/Scripts/$ENTRY"; then
  echo "Could not write the Scripts entry. Re-run this installer to repair it."
  exit 1
fi
chmod +x "$CARD/Scripts/$ENTRY"
sync
echo "Installed. Open Scripts > MisterZine-NeoGeo-TEST."
echo "Normal MisterZine and Update All settings were not changed."
if [ -d "$BACKUP" ]; then echo "Previous test installation saved at $BACKUP"; fi
echo "Press a key to return to Menu (or wait 15 seconds)."
read -r -n 1 -t 15 || true
