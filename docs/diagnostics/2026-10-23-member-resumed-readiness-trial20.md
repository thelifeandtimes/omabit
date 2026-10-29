# Replacement-readiness fixture request collision

VM20 failed its new readiness gate after the first replacement completed. The
fixture reused actor/request `qualification` / `observe-vm-zod`, already bound to
initial readiness. The runner correctly refused different input under that ID
before admitting a replacement-readiness operation. The extracted schema33 database
contains zero replacement-readiness rows. The fixture now uses the distinct
`observe-replacement-vm-zod` request for that distinct step. This does not change
production replay semantics or advise retrying uncertain work with a new ID.

The guest powered off cleanly in1385.05s; its process exited0 and the outer driver
returned1 for the failed test. Exact scope inactivity is confirmed. Both disposable
ships stopped cleanly, and seven post-shutdown state files totaling1,016,852,492 bytes
match the independently inspected disk. All19 extracted result files,13 journal
mirrors, original disks and backing artifacts were hash/equality checked. The
first verifier attempt had an incorrect exit-observation kind spelling; both that
script/error and the corrected verification are retained.

Original disks and44 metadata files are preserved. Only verified regenerable
inspection/staging copies were removed. No readiness collection or reopen gate was
reached, so this is a failed trial rather than a readiness pass. M103's independent
first-replacement pass remains intact. The corrected fixture will combine readiness
with schema34 queued Stop in a new disposable VM. All protected user resources
remain excluded.
