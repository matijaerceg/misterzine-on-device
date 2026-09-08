#!/bin/bash
# misterzine: the release tracker on your MiSTer. Main runs this from the
# Scripts menu on the framebuffer console; the binary does everything else.
DIR=/media/fat/misterzine
BIN="$DIR/misterzine"
[ -x "$BIN" ] || { echo "misterzine: binary missing at $BIN (run update_all)"; exit 1; }
trap '"$BIN" console-restore >/dev/null 2>&1; stty sane </dev/tty2 2>/dev/null' EXIT
ARGS=()
[ -e "$DIR/debug.flag" ] && ARGS+=(--debug-http=:8195)
"$BIN" "${ARGS[@]}" "$@"
ST=$?
[ $ST -ne 0 ] && echo "misterzine exited with status $ST (see $DIR/log.txt)"
exit $ST
