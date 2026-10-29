# Project member observation — 2026-10-23

The private read-only adapter now observes an explicitly journaled subset of members
during initial Create/Start. It checks exact creation contracts, complete bounded
container/network-mode inventories including indirect joins, unchanged bridge and
anchor, engine identity, active loopback publications and boot-bound namespaces.
Both member observations are bracketed by the anchor/inventory observations. An
unknown member, missing ID, changed mount/image, restart, changed namespace, host
boot, engine, inventory or cancellation refuses the observation. No name/label
discovery can adopt a container, and initial anchor Start authority is cleared.

A consistently observed member in the old namespace is reported with its actual
namespace. It is not silently rebound to the anchor's new one. The caller still
has to assess the recorded binding and require recovery. A created container has
no running namespace. Supporting a known subset permits initial members to be
created one at a time; it does not establish full-project readiness.

Twenty-two inert observation scenarios and exact-ID guards pass, alongside anchor
regressions, the focused three-package race suite and vet. No real engine or ship
was accessed. Storage-use inventory, pill/workspace ownership, durable member
Create/Start, actual private VM behavior and ship identity/readiness remain.
Schema26 and every existing receipt are unchanged.
