#!/bin/bash
# Independent of the app's display and input initialization.
if ! command -v python3 >/dev/null; then
  echo "MisterZine removal requires Python 3, supplied with current MiSTer Linux."
  exit 1
fi
exec python3 /media/fat/misterzine/maintenance.py
