# Original schema38 writer before explicit observation persistence

A frozen1603-file original source archive reproduces a959559-byte synthetic
fixture with20 tables: one completed recovery, one later interrupted drain, four
reserved member creator operations, and their original source/mirror history.
The original Go writer runs with fixed synthetic time/IDs, private HOME/database
paths and no runtime backend. Archive-only desktop scripts were not executed.

Source SHA256 is8b363b4c25c8f256b017abbeaedb0bb5b9be5e4d85f52d53778b1df90eea692b;
fixture SHA256 is27ae5f540921aacf46ed429a10f43b75785616ac0f5a73053c2eb01b92a1138c.
Two independent generations match byte-for-byte. This prepares the upgrade gate
for a future persisted observation/continuation protocol. Active schema remains38,
reader inventory remains11–37, and no actual successor-reader refusal is claimed.
Frozen VM31 is unchanged and running separately.
