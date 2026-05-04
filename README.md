# firehose

A Go CLI for the [Firehose API](https://firehose.com) — manage taps and rules and consume the SSE stream from the terminal.

## Install

```bash
go install github.com/nbw/firehose@latest
```

Or build from source:

```bash
git clone https://github.com/nbw/firehose
cd firehose
go build -o firehose .
```

## Auth

Firehose uses two tokens:

| Token | Prefix | Env var | Used for |
| --- | --- | --- | --- |
| Management key | `fhm_` | `FIREHOSE_MANAGEMENT_KEY` | `taps` subcommands |
| Tap token | `fh_` | `FIREHOSE_TAP_TOKEN` | `rules` and `stream` |

Either env var or flag works (`--management-key`, `--tap-token`); flag wins.

```bash
export FIREHOSE_MANAGEMENT_KEY=fhm_…
export FIREHOSE_TAP_TOKEN=fh_…
```

## Usage

```bash
firehose taps list
firehose taps create --name "Brand Mentions"     # token shown ONCE
firehose taps get <id>
firehose taps update <id> --name "New Name"
firehose taps delete <id>

firehose rules list
firehose rules create --value 'title:tesla' --tag market-news
firehose rules update <id> --quality=false
firehose rules delete <id>

firehose stream                                  # JSONL on stdout
firehose stream --timeout 60 --since 5m | jq .
```

Pretty by default; `--json` for raw API output. `firehose <cmd> --help` for details.

### Exit codes

`0` success · `1` generic · `2` auth (401/403/missing token) · `3` not found · `4` validation (422) · `5` rate limit (429) · `6` network/5xx · `130` SIGINT

## Streaming examples

The `stream` subcommand is in the binary, but every consumer wants a different sink. Two starting points to copy and adapt:

- [`examples/stream.sh`](examples/stream.sh) — bash + curl, no dependencies
- [`examples/stream.go`](examples/stream.go) — standalone Go (`go run examples/stream.go`)

## Claude Code Usage (skill)

A skill at [`.claude/skills/firehose/SKILL.md`](.claude/skills/firehose/SKILL.md) teaches Claude the tap and rule subcommands so it can drive the CLI for you. Streaming is intentionally excluded — point at the examples instead.

Start claude code from the root of the repository to access the firehose skill included in the repository.

Example:

1. Set a FIREHOSE_MANAGEMENT_KEY in your environment.

```
export FIREHOSE_MANGEMENT_KEY="..."
```

2. Make a tap
```
/firehose create a tap called Japan news
```

Keep the ID and token that it outputs for later.

3. Make a rule (in the same session):

```
add a rule for Japan earth quakes
```

4. Listen for news in a new window (use the token from the tap creation):

```
FIREHOSE_TAP_TOKEN="fh_...." sh examples/stream.sh
```

## Development

```bash
go build ./...
go test ./...
```
