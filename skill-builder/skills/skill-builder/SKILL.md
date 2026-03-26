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

### Step 2.1 — Collect registry info

Ask the user:
- What container registry will they push to? (e.g. `docker.io`, `ghcr.io`)
- What is their namespace on that registry? (e.g. their Docker Hub username or GitHub org)

The profile image will be pushed as `{registry}/{namespace}/{profile-name}`.

### Step 2.2 — Collect servers

Ask the user for the list of MCP servers the skill needs. For each server:
- The server name as it appears in the MCP registry (e.g. `io.github.owner/my-server`)
- The version to use, or `latest`

Repeat until the user is done adding servers.

### Step 2.3 — Fetch registry entries

URL-encode each server name before use (replace every `/` with `%2F`).

**If the user specified a concrete version:**
```
GET https://registry.modelcontextprotocol.io/v0/servers/{encodedServerName}/versions/{version}
```

**If the user said "latest"**, list all versions and find the current one:
```
GET https://registry.modelcontextprotocol.io/v0/servers/{encodedServerName}/versions
```
Find the entry where `_meta["io.modelcontextprotocol.registry/official"].isLatest === true`. Use that entry's `server.version` for the next step.

From the response, find the entry in `packages` with `registryType: oci`. Extract:
- `identifier` — the OCI image reference with tag (e.g. `docker.io/owner/repo:v1.0.0`)
- `environmentVariables` — array of config options

If no OCI package exists for a server, inform the user and skip it.

### Step 2.4 — Resolve tags to digests

For each OCI identifier, resolve the mutable tag to an immutable manifest digest. `docker pull` handles registry authentication automatically:

```bash
docker pull {identifier}
docker inspect --format='{{index .RepoDigests 0}}' {identifier}
```

The output is `{registry}/{repository}@sha256:{manifest-digest}`. This is the pinned identifier — drop the tag entirely, keep only the `@sha256:...` form.

If the pull fails, inform the user and ask whether to skip the server or abort.

### Step 2.5 — Walk through configuration options

For each server, present its `environmentVariables` one at a time. For each option, show:
- Name
- Description
- Whether it is required (`isRequired`)
- Whether it is a secret (`isSecret`)
- Default value, if any

Ask: set a value for this option, or leave it undefined for the end-user to supply?

### Step 2.6 — Emit the profile manifest

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

### Step 2.7 — Build and push scratch OCI image

Write a `Dockerfile` next to `profile.yaml`:

```dockerfile
FROM scratch
COPY profile.yaml /profile.yaml
```

Detect the host architecture, build, and push to the registry collected in Step 2.1:

```bash
ARCH=$(uname -m)
case "$ARCH" in
  arm64|aarch64) PLATFORM="linux/arm64" ;;
  *)             PLATFORM="linux/amd64" ;;
esac

IMAGE={registry}/{namespace}/{profile-name}
docker buildx build --platform $PLATFORM -t $IMAGE --push --metadata-file /tmp/profile-build-meta.json .
DIGEST=$(jq -r '."containerimage.digest"' /tmp/profile-build-meta.json)
echo "Profile digest: ${IMAGE}@${DIGEST}"
```

`${IMAGE}@${DIGEST}` is the fully-qualified registry reference (e.g. `docker.io/myuser/my-profile@sha256:abc123...`). Use this as the `profile:` value in the skill's frontmatter. This digest is immutable and resolves from any machine.

## Phase 3 — Define skill tools

Ask the user which specific MCP tools the skill will call. These come from the servers in the profile. Format: `mcp__server-name__tool-name`. Collect the full list — these become the `restrictToolAccess` entries.

## Phase 4 — Author the skill body

Help the user write the skill's instructions: what it does, how it behaves, what tools it calls and when. The generated skill body must include a configuration step at the very top of its instructions:

> **Runtime config step (include in generated SKILL.md):**
> Before doing any real work, the skill must:
>
> **0. Check if already installed (idempotency)**
> Check `.mcp.json` in the current directory, then `~/.claude/settings.json`. If every server listed in the profile's frontmatter already has an entry in `mcpServers` in either file, skip steps 1–4 entirely and proceed directly to the skill's main instructions.
>
> **1. Unpack the profile**
> Pull the profile OCI image from the registry, then extract `profile.yaml`. Pass a dummy command so Docker accepts the create for a scratch image:
> ```bash
> docker pull {profile-digest}
> docker create --name profile-{skill-name} {profile-digest} x
> docker cp profile-{skill-name}:/profile.yaml /tmp/profile-{skill-name}.yaml
> docker rm profile-{skill-name}
> ```
> `{profile-digest}` is the fully-qualified registry reference from the skill frontmatter (e.g. `docker.io/myuser/my-profile@sha256:abc123...`). Read `/tmp/profile-{skill-name}.yaml` to get the list of servers and their config.
>
> **2. Resolve undefined config values**
> Scan each server's `config` block for entries with no value set. For each undefined entry, show:
> - Name and description
> - Whether it is required
> Prompt the user to supply a value. Treat `isSecret: true` entries sensitively (do not echo).
>
> For URL-type config values, proactively suggest `host.docker.internal` as the default hostname for services on the host machine (e.g. `http://host.docker.internal:8080`). If the user provides a value containing `localhost`, warn them and suggest correcting it.
>
> For file path config values, suggest `./filename.ext` as the default (relative to the project directory).
>
> Collect all values before proceeding.
>
> **3. Prompt for scope**
> Ask the user where to register the MCP servers. Default is **project** (`.mcp.json`):
> - **project** — `.mcp.json` in the current directory (committed, shared with team) ← default
> - **user** — `~/.claude/settings.json` (applies across all projects)
> - **local** — `.claude/settings.local.json` in the current directory (git-ignored, private)
>
> **4. Update settings**
> Read the target file from Step 3 (`.mcp.json` for project scope; create it if it does not exist). The `.mcp.json` schema only allows `mcpServers` — do not add any other top-level keys.
>
> For servers that use volume mounts: Docker creates a **directory** (not a file) if the host path does not exist. Pre-create the file before writing the config:
> ```bash
> touch ./filename.ext
> ```
> Use the path the user provided from Step 2 (default `./filename.ext`). Do not attempt to resolve or compute absolute paths.
>
> **Important:** MCP servers running in Docker do not inherit the host `env` block. Pass all config values as `-e KEY=VALUE` flags inside the `args` array — do not use the `env` field:
> ```json
> {
>   "mcpServers": {
>     "{server-name}": {
>       "command": "docker",
>       "args": [
>         "run", "--rm", "-i",
>         "-e", "OPTION_ONE=resolved-value",
>         "-e", "OPTION_TWO=other-value",
>         "{identifier}"
>       ]
>     }
>   }
> }
> ```
> Use the fully-pinned `identifier` (digest form, no tag). Write the merged result back to the target file.
>
> **5. Proceed**
> Once settings are written, continue with the skill's main instructions.

## Phase 5 — Emit the skill

Ask the user where to write the skill. Common locations:
- `~/.claude/skills/{skill-name}/SKILL.md` — user-level, available in all projects
- `.claude/skills/{skill-name}/SKILL.md` — project-level, scoped to the current project

Write a single `SKILL.md` at the chosen path:

```yaml
---
description: {description}
profile: sha256:{image-id}
restrictToolAccess:
  - mcp__server-name__tool-name
---
```

Followed by the skill body authored in Phase 4.
