#!/bin/bash
# Run once after installation to enable the main-menu entry.
BIN=/media/fat/misterzine/misterzine
echo
if [ ! -x "$BIN" ]; then
  echo "MisterZine is missing. Run Update All, then try setup again."
  exit 1
fi
if "$BIN" launcher enable; then
  echo "MisterZine is ready."
  echo "Return to the MiSTer main menu and choose MisterZine."
  echo "Setup is only needed once; the menu entry starts automatically at boot."
else
  echo "Setup could not enable the main-menu entry. See the message above."
  exit 1
fi
