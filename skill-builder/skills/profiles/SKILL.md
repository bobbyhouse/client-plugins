---
description: Creates a content-addressable OCI scratch image bundling a set of MCP servers pinned by digest. Invoked when the user asks to create a profile or bundle MCP servers into a content-addressable dependency.
---

# profiles skill

Create a profile: a YAML manifest of MCP servers pinned to immutable OCI digests, bundled into a scratch OCI image. The resulting image digest is a single content-addressable reference that guarantees tools and config are consistent with agent skill expectations.

## Workflow

Follow these steps in order. Do not skip steps.

### Step 1 — Collect servers

Ask the user for the list of MCP servers to include. For each server they want to add, ask for:
- The server name as it appears in the MCP registry (e.g. `io.github.owner/my-server`)
- The version to use, or `latest`

Repeat until the user says they are done adding servers.

### Step 2 — Fetch registry entry

For each server, call the MCP registry API:

```
GET https://registry.modelcontextprotocol.io/v0.1/servers/{serverName}/versions/{version}
```

From the response, find the package entry with `registryType: oci`. Extract:
- `identifier` — the OCI image reference with tag (e.g. `ghcr.io/owner/repo:v1.0.0`)
- `environmentVariables` — array of config options

If no OCI package exists for a server, inform the user and skip that server.

### Step 3 — Resolve tag to digest

For each OCI identifier, resolve the mutable tag to an immutable digest. Parse the identifier into registry, repository, and tag, then call:

```
GET https://{registry}/v2/{repository}/manifests/{tag}
Accept: application/vnd.oci.image.manifest.v1+json
```

Read the `Docker-Content-Digest` response header to get the `sha256:...` digest. Store the server as `{registry}/{repository}@{digest}` — drop the tag entirely.

If the digest cannot be resolved, inform the user and ask whether to skip the server or abort.

### Step 4 — Walk through configuration options

For each server, present its `environmentVariables` one at a time. For each option, show:
- Name
- Description
- Whether it is required (`isRequired`)
- Whether it is a secret (`isSecret`)
- Default value, if any

Ask the user: set a value for this option, or leave it undefined for the end-user to supply?

Collect their answers.

### Step 5 — Emit the profile manifest

Write a YAML file with the following structure:

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

### Step 6 — Build scratch OCI image

Build the manifest into a scratch OCI image using `docker buildx` or `oras`:

**Option A — oras (preferred for plain artifacts):**
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

Record the image digest from the output. Report the digest to the user.
