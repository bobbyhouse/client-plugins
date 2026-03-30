---
description: Get a fresh passage from Marcus Aurelius' Meditations not seen in the last 30 days
profile: docker.io/roberthouse224/meditations-profile@sha256:b6dfc8b5758a9d92a9f5c9e21c5f5e59fccdf926e2b864232f308dcc0caacc80
restrictToolAccess:
  - mcp__project-gutenberg-mcp__list_passages
  - mcp__project-gutenberg-mcp__get_passage
  - mcp__append-log-mcp__append
  - mcp__append-log-mcp__query
---

# Meditations

Fetch a passage from Marcus Aurelius' *Meditations* that has not been shown in the last 30 days, display it with a brief explanation, and record it in the log.

---

## Step 0 — Check if already installed (idempotency)

Check `.mcp.json` in the current directory, then `~/.claude/settings.json`. If both `project-gutenberg-mcp` and `append-log-mcp` already have entries in `mcpServers` in either file, skip Steps 1–4 and go directly to Step 5.

---

## Step 1 — Unpack the profile

```bash
docker pull docker.io/roberthouse224/meditations-profile@sha256:b6dfc8b5758a9d92a9f5c9e21c5f5e59fccdf926e2b864232f308dcc0caacc80
docker create --name profile-meditations docker.io/roberthouse224/meditations-profile@sha256:b6dfc8b5758a9d92a9f5c9e21c5f5e59fccdf926e2b864232f308dcc0caacc80 x
docker cp profile-meditations:/profile.yaml /tmp/profile-meditations.yaml
docker rm profile-meditations
```

Read `/tmp/profile-meditations.yaml` to get the list of servers and their config.

---

## Step 2 — Resolve undefined config values

Scan each server's `config` block for entries with no value set. The only undefined entry is:

**`GUTENBERG_BASE_URL`** *(required)*
"Base URL of your Gutenberg mirror. Run the mirror from the mirror/ directory and point this at it."
Example: `http://localhost:8080`

Prompt the user to supply a value. If they provide a URL containing `localhost`, warn them that Docker containers cannot reach the host via `localhost` and suggest `http://host.docker.internal:<port>` instead.

---

## Step 3 — Prompt for scope

Ask the user where to register the MCP servers. Default is **project** (`.mcp.json`):

- **project** — `.mcp.json` in the current directory (committed, shared with team) ← default
- **user** — `~/.claude/settings.json` (applies across all projects)
- **local** — `.claude/settings.local.json` in the current directory (git-ignored, private)

---

## Step 4 — Update settings

Read the target file from Step 3 (create it if it does not exist). The `.mcp.json` schema only allows `mcpServers` — do not add any other top-level keys.

For each server listed in `/tmp/profile-meditations.yaml`:
1. Use the server's `name` field as the `mcpServers` key.
2. Set `command` to `"docker"`.
3. Build `args` starting with `["run", "--rm", "-i"]`, then append one `"-e", "KEY=value"` pair for every entry in the server's `config` block — using the profile value if set, or the value the user supplied in Step 2 if it was undefined. End `args` with the server's `identifier` field as the image reference.
4. For `append-log-mcp`, the log file must be volume-mounted so it persists on the host.
   Add `-v`, `/Users/bobby/log:/data` to `args` (before the image reference).
   Also add `-e`, `APPEND_LOG_FILE=/data/append-log.jsonl` so the server writes to the mounted volume instead of its working directory.

**Important:** do not use the `env` field — Docker containers do not inherit it. All values must be passed as `-e KEY=VALUE` inside `args`.

Write the merged result back to the target file.

---

## Step 5 — Fetch a fresh passage

### 5.1 — Query the log for recently seen passages

Use `mcp__append-log-mcp__query` to retrieve log entries from the last 30 days. Extract the list of passage IDs (or identifiers) already shown.

### 5.2 — List available passages

Use `mcp__project-gutenberg-mcp__list_passages` to get the full list of passages for book `2680` (Marcus Aurelius' *Meditations*).

### 5.3 — Select a fresh passage

Find a passage whose identifier does not appear in the recent log entries. If all passages have been seen in the last 30 days, inform the user and pick the least-recently-seen one.

### 5.4 — Fetch the passage text

Use `mcp__project-gutenberg-mcp__get_passage` to retrieve the full text of the selected passage.

---

## Step 6 — Display the passage

Show the passage as a formatted block quote, followed by 2–3 sentences of plain-language explanation: what Aurelius is saying, and why it matters in everyday terms. Keep the explanation brief and grounded — no purple prose.

---

## Step 7 — Record in the log

Use `mcp__append-log-mcp__append` to write a timestamped JSON entry to the log:

```json
{
  "passage_id": "<identifier>",
  "book_id": "2680",
  "shown_at": "<ISO 8601 timestamp>"
}
```
