# skill-builder plugin

A Claude Code plugin for building skills with content-addressable MCP server dependencies. It exposes one slash command:

- `/skill-builder` — end-to-end: define a skill, create a profile, emit the plugin directory

## Concept

A **profile** is a YAML manifest of MCP servers pinned to immutable OCI digests, bundled into a scratch OCI image. The image digest is the content-addressable reference — a single pointer that guarantees a fixed set of servers at fixed versions with fixed configuration.

A **skill** references a profile digest in its `SKILL.md` frontmatter. The digest is the skill's declared dependency. At invocation time, the skill reads the profile manifest to discover which servers it needs and prompts the user for any undefined configuration values.

**Why content-addressable?**
- One digest = one known-good set of servers and config
- Digests are immutable — the same digest always refers to the same artifact
- `restrictToolAccess` in the frontmatter further pins which specific tools from those servers the skill may call

## Plugin Structure

```
skill-builder/
├── .claude-plugin/
│   └── plugin.json
├── skills/
│   └── skill-builder/
│       └── SKILL.md       # /skill-builder — end-to-end skill creation
└── NOTES.md               # this file
```

## plugin.json

```json
{
  "name": "skill-builder",
  "version": "1.0.0",
  "description": "Build Claude Code skills with content-addressable MCP server dependencies"
}
```

## MCP Registry API

Base URL: `https://registry.modelcontextprotocol.io`

Fetch a server version:
```
GET /v0/servers/{serverName}/versions/{version}
```

Use `latest` as the version to get the most recent release. The response contains a `packages` array. Find the entry with `registryType: oci` — it has:
- `identifier`: the OCI image reference with a mutable tag, e.g. `ghcr.io/owner/repo:v1.0.0`
- `environmentVariables`: array of config options, each with `name`, `description`, `isRequired`, `isSecret`, and optional `default`

## Digest Resolution

The registry only stores a version tag (mutable). To pin to an immutable digest, pull the image and inspect the repo digest:

```bash
docker pull {identifier}
docker inspect --format='{{index .RepoDigests 0}}' {identifier}
```

The output is `{registry}/{repository}@sha256:{manifest-digest}`. The profile stores the server as `identifier@sha256:...` — tag dropped, digest only.

## YAML Structures

**Server options** (from registry `environmentVariables`):
```yaml
name: io.github.owner/my-mcp-server
identifier: ghcr.io/owner/my-mcp-server@sha256:abc123...
options:
  - name: CACHE_DIR
    description: Directory for caching responses
    required: false
    secret: false
    default: /tmp/cache
  - name: API_KEY
    description: API key for upstream service
    required: true
    secret: true
```

**Profile manifest** (the OCI artifact content):
```yaml
name: my-profile
servers:
  - name: io.github.owner/my-mcp-server
    identifier: ghcr.io/owner/my-mcp-server@sha256:abc123...
    config:
      CACHE_DIR: /tmp/myprofile
      API_KEY:                    # undefined — end-user supplies this
```

**Generated skill frontmatter**:
```yaml
---
description: What this skill does and when to invoke it.
profile: ghcr.io/owner/profiles@sha256:abc123...
restrictToolAccess:
  - mcp__server-name__tool-name
  - mcp__server-name__other-tool
---
```

## Generated Skill Runtime Behavior

Every skill produced by `/skill-builder` includes a mandatory configuration step at the top of its instructions. When invoked, the skill must:

1. Pull `profile.yaml` from the profile OCI image referenced in the frontmatter digest
2. Scan the manifest for `config` entries with no value set
3. For each undefined parameter, prompt the user interactively (treat `isSecret: true` entries sensitively)
4. Proceed with the skill's main instructions once all required parameters are supplied

This ensures the skill is never invoked with unconfigured servers.

## Open Questions

- How does Claude Code resolve and load a profile at skill invocation time? (explicitly deferred — not solved here)

## Related

- Claude Code plugin docs: https://code.claude.com/docs/en/plugins.md
- Claude Code skills docs: https://code.claude.com/docs/en/skills.md
- OCI image spec: https://github.com/opencontainers/image-spec
- MCP registry API reference: https://registry.modelcontextprotocol.io/docs
- MCP registry server.json schema: https://github.com/modelcontextprotocol/registry/blob/main/docs/reference/server-json/generic-server-json.md
