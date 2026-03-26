---
description: Get a fresh passage from Wuthering Heights — one not seen in the last 30 days — and display it
profile: sha256:1a2261f9c56002425fbfb546bdf12a034fd1a93008586c11b99c8553b4a7bac1
restrictToolAccess:
  - mcp__project-gutenberg-mcp__list_passages
  - mcp__project-gutenberg-mcp__get_passage
  - mcp__append-log-mcp__query
  - mcp__append-log-mcp__append
---

# wurthering

Display a passage from Wuthering Heights that hasn't been shown in the last 30 days, then log it.

---

## Step 0 — Check if already installed (idempotency)

Check `.mcp.json` in the current directory, then `~/.claude/settings.json`. If **both** of the following keys are already present in `mcpServers` in either file, skip Steps 1–4 and go straight to **Main Logic**:
- `io.github.bobbyhouse/project-gutenberg-mcp`
- `io.github.bobbyhouse/append-log-mcp`

---

## Step 1 — Unpack the profile

Extract `profile.yaml` from the profile OCI image. Pass a dummy command so Docker accepts the create for a scratch image:

```bash
docker create --name profile-wurthering sha256:1a2261f9c56002425fbfb546bdf12a034fd1a93008586c11b99c8553b4a7bac1 x
docker cp profile-wurthering:/profile.yaml /tmp/profile-wurthering.yaml
docker rm profile-wurthering
```

Read `/tmp/profile-wurthering.yaml` to confirm the server list.

---

## Step 2 — Resolve undefined config values

One config value is undefined and must be supplied by the user:

**`GUTENBERG_BASE_URL`** (required)
- Description: The base URL of a Project Gutenberg mirror (e.g. `https://www.gutenberg.org`)
- Prompt the user: *"Enter the Project Gutenberg base URL:"*

Store the supplied value for use in Step 4.

---

## Step 3 — Choose scope

Ask the user where to register the MCP servers. Default is **project**:
- **project** — `.mcp.json` in the current directory ← default
- **user** — `~/.claude/settings.json`
- **local** — `.claude/settings.local.json`

---

## Step 4 — Register MCP servers

Pre-create the log file so Docker mounts a file rather than a directory:

```bash
touch ./append-log.jsonl
```

Read the target file from Step 3 (create `.mcp.json` if it doesn't exist). Merge in the following two entries and write the result back. Use the fully-pinned digest identifiers — no tags.

```json
{
  "mcpServers": {
    "io.github.bobbyhouse/project-gutenberg-mcp": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "GUTENBERG_BASE_URL=<value from Step 2>",
        "-e", "GUTENBERG_TOOLS=list_passages,get_passage",
        "-e", "GUTENBERG_BOOK_ID=768",
        "roberthouse224/project-gutenberg-mcp@sha256:6460cba7b27343be72a85cbf5484e024711eb3aa824a18b69e67fe906722aa8d"
      ]
    },
    "io.github.bobbyhouse/append-log-mcp": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-v", "./append-log.jsonl:/data/log.jsonl",
        "-e", "APPEND_LOG_TOOLS=append,query",
        "-e", "APPEND_LOG_FILE=/data/log.jsonl",
        "roberthouse224/append-log-mcp@sha256:5008d346d8c653caab0309999e694a2a63cf3b2c3bcb7127a1bd492ed074476f"
      ]
    }
  }
}
```

Tell the user the servers have been registered and that they may need to restart Claude Code for MCP changes to take effect.

---

## Main Logic

### 1. Query recent history

Call `mcp__append-log-mcp__query` with `since_days: 30`. Extract the `passage_key` field from each logged entry to build a set of recently-seen passage keys.

### 2. List all passages

Call `mcp__project-gutenberg-mcp__list_passages` (no `book_id` argument — the server is configured with `GUTENBERG_BOOK_ID=768`). This returns a list of passage keys for Wuthering Heights.

### 3. Pick an unseen passage

Find the first passage key not in the recently-seen set. If **all** passages have been seen in the last 30 days, pick the passage whose `logged_at` timestamp is the oldest (least recently seen).

### 4. Fetch the passage

Call `mcp__project-gutenberg-mcp__get_passage` with the chosen key. The key format is `{book_id}:{byte_offset}` (e.g. `768:4096`).

### 5. Display the passage

Present the passage text to the user, formatted as a block quote. Include a small header showing the passage key for reference.

### 6. Log the passage

Call `mcp__append-log-mcp__append` with a JSON payload:

```json
{
  "passage_key": "<the key used>",
  "skill": "wurthering"
}
```

This records the passage so it won't be shown again for 30 days.
