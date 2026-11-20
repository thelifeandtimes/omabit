# Local authenticated HTTP MCP

This transport remains **experimental in0.5.0**. Source/SDK protocol checks do not
establish an installed named-client deployment. Stdio is the release's primary
MCP transport; remote OAuth/TLS service deployment is outside this release.

The development CLI serves the same typed core over Streamable HTTP:

```sh
omabit mcp create-http-token --output /ABS/PRIVATE/DIR/omabit-http.json
omabit --config-dir /ABS/REGISTRY mcp serve --transport http \
  --listen 127.0.0.1:38217 --token-file /ABS/PRIVATE/DIR/omabit-http.json \
  --allow-runner DEV:RUNNER_ID
```

The credential parent must already exist as a physical directory without group/
other write access. Creation writes a new0600 owner-only file and prints only its
path/expiration. It never replaces an existing file or prints the secret. The
credential lasts eight hours. Configure an HTTP client's protected Authorization
header from that file through the client's own secret mechanism. Keep its contents
out of command arguments, URLs, transcripts, shared configuration and source control.
A running server pins that file and rechecks its ownership/mode/inode/bytes on
every request. Rename, deletion, editing, exposure or expiry refuses subsequent
requests, including existing sessions. Rotation uses a new credential and an
explicit server restart; a session ID alone is never authentication.

The server runs in the foreground and reports its actual `/mcp` URL on stderr.
Port0 chooses a temporary local port. Stdio remains the default. HTTP binds only
a numeric loopback address and accepts the matching exact Host plus a loopback
peer. Browser Origin headers, cookies, query parameters, ambiguous authentication,
foreign protocol headers and alternate paths refuse. No TLS/proxy/client profile,
user service, enrollment, pairing or runner install is configured. These local
boundary choices follow the [MCP transport security requirements](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports).

This is a local **manual-bearer** deployment for an already configured owner/client.
It does not implement OAuth authorization-server discovery, registration, grants
or a public remote resource server. Those require a separately qualified
[OAuth resource/authorization deployment](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization).
Do not infer remote ChatGPT/provider compatibility from this local transport gate.
The [pinned official SDK authentication layer](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/auth/auth.go)
checks token expiry and binds authenticated identity to sessions; Omabit's verifier
adds exact private-file pinning and fixed, secret-free authentication failures.

Read-only tools are the default. An empty runner allowlist denies every live
target. `--enable-mutations` and explicit `--allow-source` roots retain their
existing meaning; HTTP grants no extra runner role, policy or source authority.
All calls carry the same exact target/operation bindings, timeouts, typed results
and lost-reply lookup rules as CLI/stdio. Explicit original-recording reads can
contain private data. Neither authentication nor derived redaction publishes or
authorizes sharing it. Runner-owned accepted work survives endpoint/client exit.

HTTP negotiates2025-03-26,2025-06-18 and2025-11-25 with the pinned SDK. It uses
JSON POST responses, authenticated DELETE and405 for standalone GET because this
deployment sends no unsolicited events. Stdio separately retains2024-11-05.
The newer2026 per-request protocol is refused. SDK sessions expire after five
idle minutes. Limits are16sessions,16concurrent requests,32connections,8KiB
headers, bounded request bodies, and the shared per-operation result/deadline
limits. Shutdown closes HTTP sessions without issuing runner cancellation.

Source race gates exercise all three HTTP protocol revisions, shared discovery,
runner-allowlist refusal, credential revocation/ownership/framing/concurrency and
actual CLI token creation/startup/discovery/clean shutdown. This is official-SDK
interop with inert operation fixtures. Actual named-client/provider connections,
remote OAuth/TLS deployment and installed artifact qualification remain separate.
No existing client configuration or persistent service is changed by these tests.
