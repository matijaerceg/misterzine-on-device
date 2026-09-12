#!/bin/bash
# Opens MisterZine from a Scripts menu, for a card whose main menu is another
# frontend (Degauss lists Scripts; it does not list MisterZine's menu entry).
# From MiSTer's own main menu, choose MisterZine there instead.
DIR=/media/fat/misterzine
if [ ! -x "$DIR/launch.sh" ]; then
  echo "MisterZine is missing. Run Update All, then try again."
  exit 1
fi
cd "$DIR" || exit 1
exec "$DIR/launch.sh"
