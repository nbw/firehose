---
name: firehose
description: Use when working with the Firehose CLI to manage taps and rules in this project, or when the user mentions creating, listing, updating, or deleting taps or rules. Skip for streaming — point users at examples/stream.sh or examples/stream.go.
---

# Firehose CLI

Wraps `https://api.firehose.com`. Two-tier auth: a **management key** owns one or more **taps**, each tap owns up to 25 **rules** (Lucene queries). The CLI binary is `firehose` (build with `go build -o firehose .` from the repo root).

## Auth

| Token | Prefix | Env var | Used for |
| --- | --- | --- | --- |
| Management key | `fhm_` | `FIREHOSE_MANAGEMENT_KEY` | `taps` subcommands |
| Tap token | `fh_` | `FIREHOSE_TAP_TOKEN` | `rules` subcommands, `stream` |

Either env var or flag works (`--management-key`, `--tap-token`). Flag wins. **Management keys cannot manage rules; tap tokens cannot manage taps** — this is the most common confusion.

## Taps (require management key)

```bash
firehose taps list
firehose taps create --name "Brand Mentions"   # token shown ONCE — save it
firehose taps get <id>
firehose taps update <id> --name "New Name"
firehose taps delete <id>
```

`taps create` prints the new tap token to stdout and a "shown only once" warning to stderr. The token is also retrievable later via `firehose taps list`.

## Rules (require tap token)

```bash
firehose rules list
firehose rules create --value 'title:tesla' --tag market-news
firehose rules create --value 'ahrefs OR semrush' --tag seo --quality=false
firehose rules create --value - --tag complex < query.txt   # value from stdin
firehose rules get <id>
firehose rules update <id> --tag new-tag
firehose rules update <id> --quality=false                   # disable a bool
firehose rules delete <id>
```

- `--value` is a Lucene query. Full syntax in `firehose-api.md` (indexed fields, boolean ops, wildcards, regex, date ranges).
- `--value -` reads from stdin to dodge shell-quoting issues.
- `--nsfw` defaults false, `--quality` defaults true. On `update`, only flags you set are sent — to explicitly disable a bool, use `--nsfw=false` / `--quality=false`.
- Cap: 25 rules per tap. Hitting it returns 422.

## Output and exit codes

Pretty by default, `--json` for raw API JSON (scriptable). Errors go to stderr.

| Exit | Meaning |
| ---: | --- |
| 0 | Success |
| 1 | Generic / usage |
| 2 | Auth (401, 403, missing token) |
| 3 | Not found (404) |
| 4 | Validation (422 — bad query, rule cap) |
| 5 | Rate limit (429) |
| 6 | Network / 5xx |
| 130 | Ctrl-C |

## Not covered: streaming

`firehose stream` exists but every consumer wants a different sink. Point users at:

- `examples/stream.sh` — bash + curl, copy-paste starting point
- `examples/stream.go` — standalone Go consumer (`go run examples/stream.go`)
