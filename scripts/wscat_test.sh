#!/usr/bin/env bash
# Quick manual test script using wscat (npm install -g wscat) to exercise
# auth -> subscribe -> publish flow against a local clipsync v2 server.
# Usage: ./scripts/wscat_test.sh ws://127.0.0.1:8080/v2/ws

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <ws-url>"
  exit 2
fi
URL="$1"

cat <<'EOF'
Manual wscat test steps (interactive):
1) Open first terminal and run:
   wscat -c "$URL"
   then paste:
   {"type":"auth","id":"1","body":{"token":"dev-token","client_id":"client-A"}}
   then:
   {"type":"subscribe","id":"2","body":{"channel":"office"}}

2) Open second terminal and run:
   wscat -c "$URL"
   then paste:
   {"type":"auth","id":"1","body":{"token":"dev-token","client_id":"client-B"}}
   then:
   {"type":"publish","id":"3","body":{"channel":"office","message":{"message_id":"m1","timestamp":"2026-07-01T00:00:00Z","origin":"client-B","text":"hello from B"}}}

Expect: client-A receives the publish envelope (client-B will not receive its own publish because echo is excluded).
EOF
