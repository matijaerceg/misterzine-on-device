#!/bin/bash
# Dev loop against the MisterPi. Usage: tools/dev.sh <cmd> [args]
#   build      cross-compile dist/misterzine
#   deploy     scp the binary + wrapper to the Pi (never over a running binary)
#   launch     start the Scripts entry through the Remote API
#   menu       return the Pi to the menu core
#   shot NAME  save the live canvas from the debug server to NAME.png
#   key NAME   inject one key (also: hold=ms count=n via env HOLD/COUNT)
#   keys JSON  inject a key script, e.g. '[{"key":"down","count":5}]'
#   goto KEY   put the cursor on a row key
#   state      debug state JSON
#   stack      goroutine dump from the running binary
#   logs [N]   tail the device log
#   kill       kill the binary on the Pi and restore the console
#   restore    console-restore only
#   anykey     satisfy Main's "Press any key" (Remote keyboard-raw 28)
set -e
cd "$(dirname "$0")/.."
export PATH="/c/Users/user/AppData/Local/Programs/go/bin:$PATH"
PI=${PI:-192.168.1.96}
SSH="ssh -T -o RemoteCommand=none -o RequestTTY=no -o BatchMode=yes root@$PI"
DBG="http://$PI:8195"
REMOTE="http://$PI:8182"
case "$1" in
  build)
    V=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
    GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X github.com/matijaerceg/misterzine-on-device/internal/buildinfo.Version=$V -X github.com/matijaerceg/misterzine-on-device/internal/buildinfo.Commit=$(git rev-parse --short HEAD) -X github.com/matijaerceg/misterzine-on-device/internal/buildinfo.Date=$(date -u +%Y-%m-%d)" \
      -o dist/misterzine ./cmd/misterzine
    ls -la dist/misterzine ;;
  deploy)
    $SSH "mkdir -p /media/fat/misterzine"
    scp -q dist/misterzine root@$PI:/media/fat/misterzine/misterzine.new
    scp -q deploy/Scripts/misterzine.sh root@$PI:/media/fat/Scripts/misterzine.sh
    $SSH "mv /media/fat/misterzine/misterzine.new /media/fat/misterzine/misterzine; chmod +x /media/fat/misterzine/misterzine /media/fat/Scripts/misterzine.sh; sync; /media/fat/misterzine/misterzine --version" ;;
  launch)
    curl -s "$REMOTE/api/scripts/list" | python -c "import sys,json; print('canLaunch', json.load(sys.stdin)['canLaunch'])"
    curl -s -o /dev/null -w "launch: HTTP %{http_code}\n" -X POST "$REMOTE/api/scripts/launch/misterzine.sh" ;;
  menu) curl -s -o /dev/null -w "menu: HTTP %{http_code}\n" -X POST "$REMOTE/api/launch/menu" ;;
  shot) curl -sf -o "${2:-shot}.png" "$DBG/api/shot.png" && echo "wrote ${2:-shot}.png" ;;
  key) curl -sf -X POST "$DBG/api/key/$2?hold=${HOLD:-40}&count=${COUNT:-1}" ;;
  keys) curl -sf -X POST -d "$2" "$DBG/api/keys" ;;
  goto) curl -sf "$DBG/api/goto?k=$2" ;;
  state) curl -sf "$DBG/api/state" ;;
  stack) curl -sf "$DBG/api/stack" ;;
  logs) curl -sf "$DBG/api/log?tail=${2:-60}" || $SSH "tail -n ${2:-60} /media/fat/misterzine/log.txt" ;;
  kill) $SSH "killall misterzine 2>/dev/null; sleep 0.5; /media/fat/misterzine/misterzine console-restore; echo killed" ;;
  restore) $SSH "/media/fat/misterzine/misterzine console-restore; echo restored" ;;
  anykey) curl -s -o /dev/null -w "anykey: HTTP %{http_code}\n" -X POST "$REMOTE/api/controls/keyboard-raw/28" ;;
  *) sed -n 2,16p "$0" ;;
esac
