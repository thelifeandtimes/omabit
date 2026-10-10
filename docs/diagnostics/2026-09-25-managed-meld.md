# Managed meld acceptance, September 25

The agent executed the bounded 6C live sequence through the named `workstation`
connection, pinning runner ID `runner-f496feed1f7b08f31a5d82b94ac2f5fd` on each CLI
invocation. Hello advertised `offline-meld` in `development-managed-fake` mode,
with operator actor `ssh-key:SHA256:eVDrU/23cFq0QD1D5uuM10niBgNueT+kVji4wXdJNYc`.

Preflight observed alpha healthy/running at revision 2/2 with no active operations.

| Request | Operation | Created (UTC) | Finished (UTC) | Status |
| --- | --- | --- | --- | --- |
| `checkpoint-6c-stop-alpha` | `op-535b922995c042d728c69b0d772420ac` | 18:18:31.160367069 | 18:18:31.528556346 | succeeded |
| `checkpoint-6c-meld-alpha` | `op-6b7d55049e0aefde1b2692aef84fcec8` | 18:18:56.645754350 | 18:19:17.238264128 | succeeded |
| `checkpoint-6c-resume-alpha` | `op-c4146fe1bc519205ee2908f990db224e` | 18:19:38.824432939 | 18:19:44.767176395 | succeeded |

Meld's end-to-end operation took **20.59 seconds**; resume took **5.94 seconds**.
The maintenance measurement interval was 18:18:57.222598767 through
18:19:17.237018886 UTC. Admission observed **6,469,140,480 bytes available RAM** and
**445,596,528,640 bytes free disk**.

Logical regular-file bytes changed from **640,995,340** to **627,544,076**:
**13,451,264 bytes (12.83 MiB; 2.10%)** smaller. This is not allocated disk-space
reclamation or loom utilization. The maintenance interval includes utility
lifecycle, removal, and post-measurement work.

After successful meld, the exact original request was replayed while alpha was
stopped. The returned operation matched its recorded terminal result, including
ID, actor, revision, and timestamps. After resume, the same request was replayed
again and compared with that result; it was unchanged. No second meld was created.

The final snapshot reported alpha running/healthy at revision 2/2 on
`http://127.0.0.1:18081`, without a recovery requirement or incidents. A separate
direct, proxy-free HTTP request to `/~/host` returned `~zod`.

This accepts managed meld/replay/same-pier resume. Management-restart replay and
interruption quarantine were fixture-validated; this live sequence did not induce
a destructive failure or restart management between replay checks.
