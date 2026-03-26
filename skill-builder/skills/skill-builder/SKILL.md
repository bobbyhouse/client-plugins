---
description: Guides the user through creating a new Claude Code skill with content-addressable MCP server dependencies. Invoked when the user wants to build a skill, create a skill, or author a new slash command.
---

# skill-builder skill

Create a new Claude Code skill that declares its MCP server dependencies via a content-addressable profile. The result is a ready-to-use plugin directory containing a `SKILL.md` with a pinned profile digest and `plugin.json`.

If the user already has a profile digest, skip Phase 2 and go straight to Phase 3.

## Phase 1 — Collect skill intent

Ask the user:
- What is the skill name? (used as the slash command, e.g. `my-skill` → `/my-skill`)
- What does the skill do? (one-sentence description)
- What should the skill actually accomplish? (detailed intent — used to author the skill body)

## Phase 2 — Create a profile

Follow these steps to build a content-addressable profile for the skill's MCP server dependencies.

### Step 2.1 — Collect servers

Ask the user for the list of MCP servers the skill needs. For each server:
- The server name as it appears in the MCP registry (e.g. `io.github.owner/my-server`)
- The version to use, or `latest`

Repeat until the user is done adding servers.

### Step 2.2 — Fetch registry entries

For each server, call the MCP registry API:

```
GET https://registry.modelcontextprotocol.io/v0.1/servers/{serverName}/versions/{version}
```

Find the package entry with `registryType: oci`. Extract:
- `identifier` — the OCI image reference with tag (e.g. `ghcr.io/owner/repo:v1.0.0`)
- `environmentVariables` — array of config options

If no OCI package exists for a server, inform the user and skip it.

### Step 2.3 — Resolve tags to digests

For each OCI identifier, resolve the mutable tag to an immutable digest. Parse into registry, repository, and tag, then call:

```
GET https://{registry}/v2/{repository}/manifests/{tag}
Accept: application/vnd.oci.image.manifest.v1+json
```

Read the `Docker-Content-Digest` response header to get the `sha256:...` digest. Store the server as `{registry}/{repository}@{digest}` — drop the tag entirely.

If a digest cannot be resolved, inform the user and ask whether to skip the server or abort.

### Step 2.4 — Walk through configuration options

For each server, present its `environmentVariables` one at a time. For each option, show:
- Name
- Description
- Whether it is required (`isRequired`)
- Whether it is a secret (`isSecret`)
- Default value, if any

Ask: set a value for this option, or leave it undefined for the end-user to supply?

### Step 2.5 — Emit the profile manifest

Write a YAML file:

```yaml
name: {profile-name}
servers:
  - name: {server-name}
    identifier: {registry}/{repository}@{digest}
    config:
      OPTION_ONE: value          # set by profile
      OPTION_TWO:                # undefined — end-user supplies this
```

Ask the user what to name the profile and where to write the file before writing it.

### Step 2.6 — Build scratch OCI image

Build the manifest into a scratch OCI image using `oras` or `docker buildx`:

**Option A — oras (preferred):**
```bash
oras push {registry}/{repository}:{tag} profile.yaml:application/yaml
```

**Option B — docker with scratch base:**
```dockerfile
FROM scratch
COPY profile.yaml /profile.yaml
```
```bash
docker buildx build --platform linux/amd64 -t {image-ref} --output type=oci,dest=profile.tar .
```

Record the image digest from the output. This digest is the profile's content-addressable reference.

## Phase 3 — Define skill tools

Ask the user which specific MCP tools the skill will call. These come from the servers in the profile. Format: `mcp__server-name__tool-name`. Collect the full list — these become the `restrictToolAccess` entries.

## Phase 4 — Author the skill body

Help the user write the skill's instructions: what it does, how it behaves, what tools it calls and when. The generated skill body must include a configuration step at the very top of its instructions:

> **Runtime config step (include in generated SKILL.md):**
> Before doing any real work, the skill must:
> 1. Pull `config.yaml` from the profile OCI image referenced in the frontmatter digest
> 2. Scan the manifest for `config` entries with no value set
> 3. For each undefined parameter, show the name and description and prompt the user to supply a value; treat `isSecret: true` entries sensitively
> 4. Once all required parameters are supplied, proceed with the skill's instructions

## Phase 5 — Emit skill files

Write the plugin directory with two files.

**SKILL.md frontmatter:**
```yaml
---
description: {description}
profile: {registry}/{repository}@{digest}
restrictToolAccess:
  - mcp__server-name__tool-name
---
```

**plugin.json:**
```json
{
  "name": "{skill-name}",
  "version": "1.0.0",
  "description": "{description}"
}
```

Ask the user where to write the plugin directory before writing it.
