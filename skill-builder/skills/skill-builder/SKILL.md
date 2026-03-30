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

Ask: set a value for this option, or leave it for the end-user to supply at invocation time?

If the user sets a value, it goes into the profile as a literal string. If they leave it for the end-user, write it as a `${KEY}` placeholder — the gateway will require the caller to supply that key via the `config` argument on `load`.

**Required env vars (`isRequired: true`) with no default value must never be omitted from the profile.** If the user does not supply a literal value, it must become a `${KEY}` placeholder. Silently dropping a required var will cause the container to crash on startup.

Then ask: does this server need any host paths mounted into the container? For each mount, collect a `host:container` path pair. Host paths may use `${KEY}` placeholders — the gateway resolves them the same way it resolves config values, so the caller must supply the key via `config` on `load`.

### Step 2.6 — Emit the profile manifest

Write a YAML file:

```yaml
name: {profile-name}
servers:
  - name: {server-name}
    identifier: {registry}/{repository}@{digest}
    config:
      OPTION_ONE: value          # set by profile
      OPTION_TWO: ${OPTION_TWO}  # end-user supplies this via invoke config
    mounts:
      - /host/path:/container/path        # literal host path
      - ${WORKSPACE_DIR}:/workspace       # end-user supplies WORKSPACE_DIR via invoke config
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

`${IMAGE}@${DIGEST}` is the fully-qualified registry reference (e.g. `docker.io/myuser/my-profile@sha256:abc123...`). Record this — it becomes the `profile` argument in the `load` call in the skill body.

## Phase 3 — Resolve the gateway load tool name

The gateway MCP server can be registered under any name — you only know that its bootstrap tool is always called `load`. Before authoring the skill, determine the fully-qualified tool name.

Look at your available tools and find the one whose name ends with `__load`. That tool name (e.g. `mcp__profile-gateway__load` or `mcp__gateway__load`) confirms the gateway is available. Record it as `{gateway-load-tool}` for use in Phase 5.

If no such tool is available, tell the user that the gateway plugin does not appear to be installed or enabled, and ask them to install it before continuing.

The generated skill must **not** hardcode this name. Instead, it must instruct the executing Claude to discover the gateway tool at runtime by finding the available tool whose name ends with `__load` and using that name for the bootstrap call.

## Phase 4 — Identify downstream tools

Ask the user which specific MCP tools the skill will call. These come from the servers in the profile. Collect the full list in `server-name__tool-name` format — they will be referenced in the skill body in Phase 5. Argument schemas do not need to be documented; they will be available to the executing Claude after the profile is loaded.

## Phase 5 — Author the skill body

Help the user write the skill's instructions: what it does, how it behaves, what tools it calls and when.

**Bootstrap pattern (use this in every generated skill):**

The skill must open with a bootstrap section that loads the profile before doing anything else. This registers the downstream MCP tools into the session so the executing Claude can call them directly with their real schemas.

Example wording:

> ## Bootstrap
>
> Before making any other tool calls, find the available tool whose name ends with `__load` and call it to load the profile:
>
> ```
> <gateway-load-tool>(
>   profile="{profile-digest}",
>   config={...any config values known at authoring time...}
> )
> ```
>
> If no such tool is available, tell the user the gateway plugin does not appear to be installed.

Where `{profile-digest}` is the fully-qualified OCI reference from Phase 2.7 — a constant baked into the skill body.

**Handling missing config (use this in every generated skill that has `${KEY}` placeholders):**

If the `load` call returns an error whose message contains "missing required config keys", the skill must:
1. Show the user which keys are missing and ask them to supply values. For URL-type keys suggest `http://host.docker.internal:<port>` for local services; warn if the user provides a value containing `localhost`. For file path keys (including mount host paths) suggest an absolute path.
2. Retry the `load` call with a `config` argument supplying the collected values.

**Calling downstream tools:**

After `load` succeeds, the downstream tools are registered in the session and available to call directly — they appear as `server-name__tool-name` in the executing Claude's tool list with their real input schemas. The skill body should describe which tools to call and what to do with their results, without specifying argument schemas (the executing Claude will have those from the registered tool definitions).

## Phase 6 — Emit the skill

Ask the user where to write the skill. Common locations:
- `~/.claude/skills/{skill-name}/SKILL.md` — user-level, available in all projects
- `.claude/skills/{skill-name}/SKILL.md` — project-level, scoped to the current project

Write a single `SKILL.md` at the chosen path:

```yaml
---
description: {description}
---
```

Followed by the skill body authored in Phase 5. Do not add `restrictToolAccess` — the gateway tool name and downstream tool names are discovered at runtime and are not known at authoring time.
