# Omabit 0.1.0-rc.6 — effective Recommended Apps default

This corrects a live integration issue found while deploying RC.5: the Omarchy
widget can receive only explicit settings, so the selector must apply its packaged
default itself. Recommended Apps now resolves to
`https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill` when the setting is
omitted. Explicit custom sources remain effective; an explicitly empty source
continues to require configuration. Bare bootstrap remains independent.

RC.6 includes all [RC.5 feedback refinements](release-rc5.md): binary/source setup
information, stable log controls, inline port progress, visible public comet names,
HTTP fleet labels/Open actions and Application Connections marked Coming Soon.

The Go backend, schema16, runtime image, resource limits and supplied pill bytes
are identical in source to RC.5. The published RC.5 release remains immutable;
RC.6 contains the corrected native/snapshot UI. The provider pill's frozen hash is
`85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62`.

Cold-reload the unlocked shell after updating and verify its live version. The
monitor status also reports whether the supplied Recommended Apps default is
effective, so installed metadata alone cannot pass that check.
