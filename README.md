# ste-check

Check text for Simplified Technical English (STE).

> [!IMPORTANT]
> This is an unofficial tool. It is not related to ASD, and it does not use the official [ASD-STE100](https://www.asd-ste100.org/) dictionary. Its word list and rules are an approximation. To make sure that a text obeys ASD-STE100, use the official specification.

```sh
ste-check "Prior to commencing, the cover must be removed."
echo "Close the valve." | ste-check
ste-check --description < notes.txt
```

Exit codes: `0` no issues, `1` issues found, `2` usage or config error. To disable color, set `NO_COLOR`.

## Rules

| Rule | Checks |
| --- | --- |
| `max-sentence-length` | More than 20 words in a procedure sentence, or 25 in a description (`-d`) |
| `no-unapproved-words` | Words and phrases from an [unofficial list](ste/words.txt), with simpler alternatives |
| `no-ing` | `-ing` verbs, except words on the [allow list](ste/ing-allowed.txt) and capitalized names |
| `no-passive` | Passive voice, such as "must be removed" |
| `no-contractions` | Contractions, such as "don't". Possessives are not flagged. |
| `no-perfect-tense` | Perfect tenses, such as "have removed" |

## Configuration

`ste-check` reads `~/.config/ste-check/config.json` if it exists, or the file you give with `--config`. To write a starter config and its schema there, run `ste-check --init`. It keeps a `config.json` that exists. Run it again after an update to get the latest schema.

```json
{
  "$schema": "./schema.json",
  "type": "description",
  "maxWords": { "procedure": 20, "description": 25 },
  "rules": { "no-contractions": false },
  "words": { "spin up": "start" },
  "allowIng": ["caching"],
  "ignore": ["Acme"]
}
```

All keys are optional. [`schema.json`](schema.json) tells what each key does, and gives autocomplete in editors such as VS Code.

## Install

Install [Go](https://go.dev/doc/install), then run:

```sh
GOBIN="$HOME/.local/bin" go install github.com/SeanMcP/ste-check@latest
```

Make sure that `~/.local/bin` is on your `PATH`. From a clone, run `GOBIN="$HOME/.local/bin" go install .` after each change.

## Develop

```sh
go test ./...
go run . "Text to check."
```

- To flag a word, add `unapproved = alternative` to `ste/words.txt`. Single words also match common forms ("verify" matches "verified").
- To allow an `-ing` word, add it in lowercase to `ste/ing-allowed.txt`. Add only words that are almost always nouns or names.
- If you add or rename a rule, also update `schema.json`.

Tests check that no alternative is itself on the list, and that the schema matches the rules.
