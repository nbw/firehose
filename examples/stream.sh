#!/usr/bin/env bash
# Minimal Firehose SSE consumer in pure bash + curl.
#
# Educational starting point — copy and adapt to your sink (file, Slack,
# downstream processor). The CLI's `firehose stream` does the same thing
# with more features (reconnect, JSONL output, exit codes).
#
# Requires: bash, curl. jq is optional (for filtering output).

set -euo pipefail

: "${FIREHOSE_TAP_TOKEN:?FIREHOSE_TAP_TOKEN is required (export it)}"
BASE_URL="${FIREHOSE_BASE_URL:-https://api.firehose.com}"
TIMEOUT="${TIMEOUT:-300}"

trap 'echo "stopped" >&2; exit 130' INT TERM

echo "connecting to ${BASE_URL}/v1/stream..." >&2

event=""
data=""

# -N disables curl's output buffer; -s silences progress; --no-buffer is curl 7.18+.
curl -N -s --no-buffer \
    -H "Authorization: Bearer ${FIREHOSE_TAP_TOKEN}" \
    -H "Accept: text/event-stream" \
    -H "Cache-Control: no-cache" \
    "${BASE_URL}/v1/stream?timeout=${TIMEOUT}" |
while IFS= read -r line; do
    # Strip trailing \r if the server sends CRLF
    line="${line%$'\r'}"

    # Blank line: dispatch the accumulated event
    if [[ -z "$line" ]]; then
        if [[ -n "$event" || -n "$data" ]]; then
            case "$event" in
                connected) echo "connected — waiting for matches (timeout=${TIMEOUT}s)" >&2 ;;
                end)       echo "stream ended" >&2 ;;
                error)     echo "stream error: $data" >&2 ;;
                *)         printf '%s\t%s\n' "$event" "$data" ;;
            esac
            event=""
            data=""
        fi
        continue
    fi

    # Comment line — ignored per SSE spec
    if [[ "${line:0:1}" == ":" ]]; then
        continue
    fi

    # Split on first colon, strip one leading space from value
    field="${line%%:*}"
    value="${line#*:}"
    value="${value# }"

    case "$field" in
        event) event="$value" ;;
        data)
            if [[ -n "$data" ]]; then
                data="${data}"$'\n'"${value}"
            else
                data="$value"
            fi
            ;;
        id) ;;
        retry) ;;
    esac
done

# Pipe to jq for pretty filtering, e.g.:
#   ./stream.sh | awk -F'\t' '$1=="update"{print $2}' | jq -r '.document.url'
