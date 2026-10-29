# Preserve foundation-v1 code hash encoding in the live probe

M410, 2026-10-26. A second read-only audit of retained VM05 finds twelve complete
request prefixes, each ending at sequence6: the first eth_getCode, for Azimuth.
All72 requests were accepted; no chain read failed. Disk/base hashes are unchanged
and the audit VM is inactive. Only fixed request methods/counts were exported.

The original deployment writer hashes the canonical lowercase, 0x-prefixed
hexadecimal code string as UTF-8. The new live probe instead decoded it to bytes
before hashing. For Azimuth those hashes are cf74d4cc… and 3df91aef… respectively.
Independent comparison of all nine pinned compiled/deployment artifacts confirms
the same encoding mismatch. The existing foundation contract remains unchanged.

The corrected probe hashes the validated code string as UTF-8. Tests now use the
original foundation convention and explicitly refuse decoded-byte hashes. They
reproduce two failures against the old source; all47 JavaScript checks pass on
the correction. All355 Go gateway race checks and package vet pass. Exact tested
bytes are integrated. No deadline, ownership check, foundation pin, input generation,
persistent schema or runtime permission was changed.

The audit projection has three synthetic tests and host/child refusal checks. Its
first standalone build was refused by Go's internal-package boundary; a package
overlay compiled successfully without repository changes. Test evidence wrappers
were corrected to request TAP in-process and select the original writer's bridge
Naive contract explicitly. Earlier attempts remain retained; no runtime mutation
was repeated.

M410 supersedes M409's unknown RPC substage and unfixed source defect only. The
failed diagnostic run remains a failure. Fresh ordinary binaries must pass managed
initialization and separately attributed recovery before runtime qualification.
