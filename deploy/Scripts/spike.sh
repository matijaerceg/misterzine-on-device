#!/bin/bash
# Phase-0 spike launcher. Args come from /media/fat/misterzine/spike.args so
# the same Scripts entry can run any experiment without editing this file.
DIR=/media/fat/misterzine
ARGS=$(cat "$DIR/spike.args" 2>/dev/null)
"$DIR/spike" $ARGS
echo "spike exited with status $? (log: $DIR/spike.log)"
