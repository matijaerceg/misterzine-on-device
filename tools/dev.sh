#!/bin/bash
# Device development loop. Usage: PI=<address> tools/dev.sh <cmd> [args]
#   build      cross-compile dist/misterzine
#   deploy     scp the binary + wrapper to the Pi (never over a running binary)
#   launch     select MisterZine through the MiSTer command interface
#   menu       return the Pi to the menu core
#   shot NAME  save the live canvas from the debug server to NAME.png
#   key NAME   inject one key (also: hold=ms count=n via env HOLD/COUNT)
#   keys JSON  inject a key script, e.g. '[{"key":"down","count":5}]'
#   goto KEY   put the cursor on a row key
#   state      debug state JSON
#   stack      goroutine dump from the running binary
#   logs [N]   tail the device log
#   quit       close the UI through its authenticated debug endpoint
#   restore    console-restore only
#   anykey     satisfy Main's "Press any key" (Remote keyboard-raw 28)
set -e
cd "$(dirname "$0")/.."
if [ "${1:-}" != build ]; then
  : "${PI:?Set PI to the device IP address or SSH hostname}"
fi
SSH="ssh -T -o RemoteCommand=none -o RequestTTY=no -o BatchMode=yes root@$PI"
DBG="http://$PI:8195"
REMOTE="http://$PI:8182"
debug_curl() {
  local token
  token=$($SSH "cat /media/fat/misterzine/debug-token")
  curl -H "X-MisterZine-Token: $token" "$@"
}
case "$1" in
  build)
    V=${VERSION:-v1.0.0-dev}
    GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X github.com/matijaerceg/misterzine-on-device/internal/buildinfo.Version=$V -X github.com/matijaerceg/misterzine-on-device/internal/buildinfo.Commit=$(git rev-parse --short HEAD) -X github.com/matijaerceg/misterzine-on-device/internal/buildinfo.Date=$(date -u +%Y-%m-%d)" \
      -o dist/misterzine ./cmd/misterzine
    ls -la dist/misterzine ;;
  deploy)
    $SSH "mkdir -p /media/fat/misterzine"
    scp -q dist/misterzine root@$PI:/media/fat/misterzine/misterzine.new
    scp -q deploy/launch.sh root@$PI:/media/fat/misterzine/launch.sh
    scp -q deploy/MisterZine.mgl root@$PI:/media/fat/MisterZine.mgl
    $SSH "mv /media/fat/misterzine/misterzine.new /media/fat/misterzine/misterzine; chmod +x /media/fat/misterzine/misterzine /media/fat/misterzine/launch.sh; sync; /media/fat/misterzine/misterzine --version" ;;
  launch)
    $SSH "timeout 3 sh -c 'echo load_core /media/fat/MisterZine.mgl > /dev/MiSTer_cmd'" ;;
  menu) curl -s -o /dev/null -w "menu: HTTP %{http_code}\n" -X POST "$REMOTE/api/launch/menu" ;;
  shot) debug_curl -sf -o "${2:-shot}.png" "$DBG/api/shot.png" && echo "wrote ${2:-shot}.png" ;;
  key) debug_curl -sf -X POST "$DBG/api/key/$2?hold=${HOLD:-40}&count=${COUNT:-1}" ;;
  keys) debug_curl -sf -X POST -d "$2" "$DBG/api/keys" ;;
  goto) debug_curl -sf -X POST "$DBG/api/goto?k=$2" ;;
  state) debug_curl -sf "$DBG/api/state" ;;
  stack) debug_curl -sf "$DBG/api/stack" ;;
  logs) debug_curl -sf "$DBG/api/log?tail=${2:-60}" || $SSH "tail -n ${2:-60} /media/fat/misterzine/log.txt" ;;
  quit|kill) debug_curl -sf -X POST "$DBG/api/quit" ;;
  restore) $SSH "/media/fat/misterzine/misterzine console-restore; echo restored" ;;
  anykey) curl -s -o /dev/null -w "anykey: HTTP %{http_code}\n" -X POST "$REMOTE/api/controls/keyboard-raw/28" ;;
  *) sed -n 2,16p "$0" ;;
esac
