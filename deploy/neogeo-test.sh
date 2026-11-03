#!/bin/bash
# MisterZine's console wrapper, opened by the main-menu launcher.
DIR=/media/fat/misterzine-neogeo-test
BIN="$DIR/mz-neogeo"
[ -x "$BIN" ] || { echo "misterzine: binary missing at $BIN (extract the Neo Geo test ZIP again)"; exit 1; }
# Preserve a restore helper even if Update All replaces/removes the installed file.
RESTORE_DIR=$(mktemp -d /tmp/misterzine-restore.XXXXXX) || exit 1
RESTORE="$RESTORE_DIR/restore"
if ! cp "$BIN" "$RESTORE"; then rm -f "$RESTORE"; rmdir "$RESTORE_DIR"; exit 1; fi
chmod +x "$RESTORE"
trap '"$RESTORE" console-restore >/dev/null 2>&1; stty sane </dev/tty2 2>/dev/null; rm -f "$RESTORE"; rmdir "$RESTORE_DIR"' EXIT
echo
echo "  MisterZine Neo Geo TEST is loading..."
ARGS=()
[ -e "$DIR/debug.flag" ] && ARGS+=(--debug-http=:8197)
"$BIN" "${ARGS[@]}" "$@"
ST=$?
# Interrupts and requested shutdowns still restore the console through EXIT.
if [ "$ST" -ne 0 ] && [ "$ST" -ne 130 ] && [ "$ST" -ne 143 ]; then
  "$RESTORE" console-restore >/dev/null 2>&1
  echo "MisterZine could not continue (status $ST). Recent log:"
  tail -n 6 "$DIR/log.txt" 2>/dev/null
  echo "Press a key to return to Menu (or wait 15 seconds)."
  read -r -n 1 -t 15 || true
fi
exit $ST
