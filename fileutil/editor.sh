#!/bin/sh

set -e

if [ -z "$1" ]
then exit 1
fi
if [ "$EDITOR_TEXT" ]
then echo "$EDITOR_TEXT" >"$1"
fi

if [ -z "$EDITOR_EXIT_STATUS" ]
then EDITOR_EXIT_STATUS=0
fi
exit "$EDITOR_EXIT_STATUS"
