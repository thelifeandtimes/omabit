# Second coordinated recovery through concrete adapters

The inert engine now qualifies a full second drain, namespace selection, retirement,
same-pier replacement and readiness. Both a stopped anchor requiring one Start and
an already restarted anchor requiring no Start pass. A mixed pair (one coordinated
creator, one later ordinary replacement) also passes. Every command is checked
against the exact committed phase/ID; both piers, peer identity and all preceding
history remain intact. A subsequent reader reconstructs the second boundary and
refuses healthy repair.

A before-fix regression proved that an old retired ID reappearing outside the
project network could fail to fence a later signal. All recovery phases now reject
historically retired IDs, reusing complete inventories where already available;
drain/namespace observations bracket those inventories when prior epochs exist.
Faults before signal, namespace adoption, removal, Create, Start and during readiness
all refuse the next relevant effect or completion. Ordinary initial paths keep
their prior inventory behavior.

Nine second-cycle scenarios pass race213.241s; related adapters pass race309.22s.
Managed compile/VM skip, vet and diff checks pass. The before-fix failure and all
source/log evidence are retained. This supersedes M160's pending second concrete
recovery/history guards only. It is not real Docker coordinated-repair evidence;
persistence and public scheduling remain absent. Ordinary post-recovery lifecycle
integration is the next source gate. VM29 remains the separate diagnostic/runtime
reference and excludes these later changes.
