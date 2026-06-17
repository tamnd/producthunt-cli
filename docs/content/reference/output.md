---
title: "Output formats"
description: "The output contract every command shares: formats, fields, and templates."
weight: 30
---

Every command renders through one formatter, so the same flags work everywhere.
Pick a format with `-o`, or let ph choose: a table when writing to a terminal, JSONL when piped.

## Formats

```bash
ph <command> -o table     # a rounded, color-aware grid for reading
ph <command> -o markdown  # a GitHub pipe table to paste into docs (alias: md)
ph <command> -o list      # one record per section, easy on the eyes
ph <command> -o jsonl     # one JSON object per line, for piping
ph <command> -o json      # a single JSON array
ph <command> -o csv       # spreadsheet friendly
ph <command> -o tsv       # tab-separated
ph <command> -o url       # just the URL column
ph <command> -o raw       # the underlying bytes, unformatted
```

| Format | Best for |
|---|---|
| `table` | Reading on a terminal: a rounded border with an accented header |
| `markdown` | Pasting into a README, issue, or PR (alias `md`) |
| `list` | Reading one record at a time: a heading and a short bullet list per record |
| `jsonl` | Piping into another tool, one object at a time |
| `json` | Loading a whole result as an array |
| `csv` / `tsv` | Spreadsheets and quick column math |
| `url` | Feeding URLs into other commands |
| `raw` | The unformatted bytes (response bodies, file contents) |

## Color

On an interactive terminal the `table`, `list`, and `json`/`jsonl` formats are colored: the table draws a dim border with an accented header, `list` styles each record's heading and keys, and JSON keys, strings, numbers, and literals are highlighted.
Color is suppressed the moment output is not a terminal, so a pipe always gets plain, parseable bytes.
Force the choice with `--color always|never` (or set `NO_COLOR`).
`markdown`, `csv`, `tsv`, `url`, and `raw` are never colored, so they stay safe to redirect into a file.

## Narrowing columns

Keep only the fields you want:

```bash
ph <command> --fields name,votes,url
```

`--no-header` drops the header row in `table` and `csv` output, which helps when a downstream tool expects bare rows.

## Templating rows

For full control over each line, apply a Go text/template.
Fields are the JSON keys, capitalised:

```bash
ph <command> --template '{{.URL}} {{.Name}}'
```

## Why auto-detection helps

Because the default adapts to the destination, the same command reads well by hand and parses cleanly in a pipe:

```bash
ph posts            # a table, because this is a terminal
ph posts | wc -l    # JSONL, because this is a pipe
```

You reach for `-o` when you want something other than that default.
