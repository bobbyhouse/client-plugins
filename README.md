# plugins

A collection of Claude Code plugins for building and managing skills with content-addressable MCP server dependencies.

## Plugins

### skill-builder

Tools for creating Claude Code skills that pin their MCP server dependencies to immutable OCI digests.

**Slash commands:**

- `/skill-builder` — End-to-end workflow: collects skill intent, creates a profile, defines tool access, authors skill instructions, and emits the plugin directory.
- `/skill-builder:profiles` — Standalone profile creation. Useful for creating or updating profiles independently of a full skill build.

A **profile** is a YAML manifest of MCP servers pinned to immutable OCI digests, bundled into a scratch OCI image. Skills reference a profile digest in their frontmatter to guarantee reproducible dependencies across any environment.

See [`skill-builder/NOTES.md`](skill-builder/NOTES.md) for technical details on profiles, skills, the MCP registry API, and OCI digest resolution.

### demos

Example skills built with skill-builder.

- `/demos:wurthering` — Fetches a random passage from *Wuthering Heights* via Project Gutenberg, avoiding passages shown in the last 2 days.

## Structure

```
plugins/
├── .claude-plugin/marketplace.json   # Plugin marketplace registration
├── skill-builder/                    # skill-builder plugin
│   ├── NOTES.md                      # Technical reference
│   └── skills/
│       ├── skill-builder/SKILL.md    # /skill-builder command
│       └── profiles/SKILL.md         # /skill-builder:profiles command
└── demos/                            # Demo skills
    └── skills/
        └── wurthering/SKILL.md       # Wuthering Heights passage fetcher
```

## MCP Servers

The repo includes two MCP servers (run as Docker containers) used by the demo skills:

| Server | Purpose |
|---|---|
| `project-gutenberg-mcp` | Fetch book metadata and passages from Project Gutenberg |
| `append-log-mcp` | Append-only JSONL log for tracking events across sessions |
