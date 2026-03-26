---
description: Fetch a random passage from Wuthering Heights via Project Gutenberg, avoiding repeats from the last 2 days.
profile: registry-1.docker.io/roberthouse224/wurthering-profile@sha256:29a1602bcf9913e48cb1b8baaa172893849e834310c52a0bbd5686bdb0bb9018
restrictToolAccess:
  - mcp__project-gutenberg-mcp__list_passages
  - mcp__project-gutenberg-mcp__get_passage
  - mcp__append-log-mcp__append
  - mcp__append-log-mcp__query
---

# wurthering skill

Fetch and display a random passage from Wuthering Heights, logging each one so repeats are avoided within a 2-day window.

## Runtime Configuration

Before doing any work:

1. Pull `profile.yaml` from the profile OCI image referenced in the frontmatter digest:
   ```
   registry-1.docker.io/roberthouse224/wurthering-profile@sha256:29a1602bcf9913e48cb1b8baaa172893849e834310c52a0bbd5686bdb0bb9018
   ```
2. Scan `profile.yaml` for `config` entries with no value set (blank entries).
3. For each undefined parameter, display its name and description and prompt the user to supply a value.
   - Currently undefined: `GUTENBERG_BASE_URL` — URL of the Project Gutenberg mirror to read from (e.g. `https://www.gutenberg.org`).
4. Once all required values are collected, proceed.

## Workflow

### Step 1 — Check recent history

Call `mcp__append-log-mcp__query` to retrieve log entries from the last 2 days (48 hours). Extract the set of passage identifiers that have already been shown. Each log entry has the shape:

```json
{
  "timestamp": "<ISO 8601>",
  "book_id": "768",
  "passage_id": "<passage identifier>"
}
```

### Step 2 — List available passages

Call `mcp__project-gutenberg-mcp__list_passages` for book ID `768` (Wuthering Heights). This returns a list of passage identifiers/indices.

### Step 3 — Select a fresh passage

From the full list of passages, remove any whose `passage_id` appears in the recent history set from Step 1.

- If fresh passages remain, pick one at random.
- If **all** passages were shown within the last 2 days, pick the one whose most recent log entry is the oldest (i.e. the least recently shown), so the user gets something different.

### Step 4 — Fetch and display the passage

Call `mcp__project-gutenberg-mcp__get_passage` with the selected passage identifier. Display the retrieved text to the user.

### Step 5 — Log the passage

Call `mcp__append-log-mcp__append` to record the shown passage:

```json
{
  "book_id": "768",
  "passage_id": "<the passage identifier used>"
}
```

The server automatically timestamps the entry.
