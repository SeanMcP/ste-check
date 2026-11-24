# ste-check

Check text for Simplified Technical English (STE).

> [!IMPORTANT]
> This is an unofficial tool. It is not related to ASD, and it does not use the official [ASD-STE100](https://www.asd-ste100.org/) dictionary. Its word list and rules are an approximation. To make sure that a text obeys ASD-STE100, use the official specification.

```sh
ste-check "Prior to commencing, the cover must be removed."
echo "Close the valve." | ste-check
ste-check --description < notes.txt
```

## Rules

| Rule | Checks |
|---|---|
| `max-sentence-length` | A maximum of 20 words in each sentence of a procedure, and 25 in a description (`-d`) |
| `no-unapproved-words` | Complex words, idioms, and phrasal verbs from an [unofficial word list](ste/words.txt), with simpler alternatives |
| `no-ing` | `-ing` verb forms, except words on the [allow list](ste/ing-allowed.txt) (like "bearing" or "padding") and capitalized names after the first word |
| `no-passive` | Passive voice, such as "must be removed" or "is opened by" |
| `no-contractions` | Contractions, such as "don't" or "it's". Possessives like "the pump's" are not flagged. |
| `no-perfect-tense` | Perfect tenses, such as "have removed" or "hasn't seen" |

Exit codes: `0` no issues, `1` issues found, `2` usage or config error. To disable color, set `NO_COLOR`.

## Configuration

`ste-check` reads `~/.config/ste-check/config.json` if it exists. To use a different file, add `--config path/to/config.json`.

To make a starter config, run:

```sh
ste-check --init
```

This writes `config.json` and `schema.json` to `~/.config/ste-check/`. It does not replace a `config.json` that already exists. After you update `ste-check`, run it again to get the latest schema.

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

| Key | Use |
|---|---|
| `type` | The default text type: `procedure` or `description`. `-p` and `-d` change it for one run. |
| `maxWords` | The maximum number of words in a sentence, for each text type |
| `rules` | To disable a rule, set it to `false`. All rules are enabled by default. |
| `words` | More words or phrases to flag, with the alternative to suggest |
| `allowIng` | More `-ing` words to allow |
| `ignore` | Words or phrases that no rule flags, such as product names |

All keys are optional. The schema gives autocomplete and error messages in editors such as VS Code. To use the schema without a local copy, set `$schema` to `https://raw.githubusercontent.com/SeanMcP/ste-check/main/schema.json`.

## Install

Install [Go](https://go.dev/doc/install) (`brew install go` on macOS). Then install the binary in a directory that is on your `PATH`:

```sh
GOBIN="$HOME/.local/bin" go install github.com/SeanMcP/ste-check@latest
```

To install from a clone of this repo, run `GOBIN="$HOME/.local/bin" go install .` in the repo directory. After you change the code, run it again.

To make sure that it worked, run `which ste-check`. If you get no output, add the directory to your `PATH` in `~/.zshrc`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

To uninstall, run `rm "$HOME/.local/bin/ste-check"`.

## Develop

```sh
go test ./...
go run . "Text to check."
```

To flag a word, add a line to `ste/words.txt` in the format `unapproved = alternative`. Single words also match common forms, so `verify` also matches "verified". The tests make sure that no alternative is itself on the list.

To allow an `-ing` word, add it in lowercase to `ste/ing-allowed.txt`. Add a word only if it is almost always a noun or a name.

If you add or rename a rule, also update `schema.json`. A test makes sure that they match.

The checker is in `ste/ste.go`.
