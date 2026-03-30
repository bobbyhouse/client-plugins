---
description: Fetch a fresh passage from Marcus Aurelius' Meditations not seen in the last 30 days, with a brief explanation.
---

# meditations

Retrieve a passage from Marcus Aurelius' *Meditations* (Gutenberg book 2680) that has not been shown in the last 30 days, then display it with a brief explanation.

## Bootstrap

Before any other tool calls, ask the user to supply the following two values (do not skip this step even if you think you know the values):

- **`GUTENBERG_BASE_URL`** — the base URL of their Gutenberg mirror (e.g. `https://www.gutenberg.org`). If running a local mirror, use `http://host.docker.internal:<port>` — do not use `localhost`.
- **`LOG_DIR`** — an absolute path to a directory on the host where the log file will be persisted (e.g. `~/.claude/meditations-log`). The directory will be created if it does not exist.

Once you have those values, find the available tool whose name ends with `__load` and call it.

If no tool ending in `__load` is available, tell the user the gateway plugin does not appear to be installed and stop.

**CRITICAL — `config` must be a JSON object, never a string.** Pass it like this (with the actual values substituted):

```json
{
  "profile": "docker.io/roberthouse224/meditations-profile@sha256:54b8a42e1a255e265479408adc0f9267b6d0241fd07a8583e5dc49695250a826",
  "config": {
    "GUTENBERG_BASE_URL": "<value supplied by user>",
    "LOG_DIR": "<value supplied by user>"
  }
}
```

Do **not** serialize `config` as a string (e.g. `"{}"` or `"{\"KEY\":\"value\"}"`) — it must be a plain JSON object.

## Fetch a fresh passage

Once `load` succeeds, the downstream tools are registered. Use them as follows:

1. **Query recent history** — call `append-log-mcp__query` to retrieve log entries from the last 30 days. Filter for entries with `skill: "meditations"`. Collect the `passage_ref` values (the byte-offset keys, e.g. `"2680:196608"`) that appear — these are off-limits.

2. **Pick a passage** — call `project-gutenberg-mcp__list_passages` to get all available keys. Randomly select one that does **not** appear in the recent history list.

3. **Retrieve the passage** — call `project-gutenberg-mcp__get_passage` with the chosen key. If the text is very short (< 20 words) or looks like a navigation artifact or biographical introduction rather than Aurelius' own words, pick a different key.

## Display

Show the passage in a clean block:

> **Meditations**
>
> *{passage text}*
>
> {one or two sentence explanation of what Aurelius is expressing and why it matters}

Keep the explanation concise — the passage should speak for itself.

## Log the passage

After displaying, call `append-log-mcp__append` to record the entry:

```json
{
  "skill": "meditations",
  "passage_ref": "{the key used, e.g. 2680:196608}",
  "shown_at": "{ISO-8601 timestamp}"
}
```
