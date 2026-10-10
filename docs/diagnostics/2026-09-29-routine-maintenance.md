# Timed routine accepted — September 29 UTC

The 6S runner advertised `maintenance-schedules` in the existing pinned
`development-managed` enrollment. All four running instances were healthy and
`import-check` had no existing policy. Available RAM was approximately 8 GiB.

`scripts/validate-routine.py` created policy
`f702d36eabb8f6a03aaed0252df8dea1`, revision 1, bound to import-check configuration
revision 1 and the authenticated SSH operator. It explicitly enabled pack on UTC
Tuesday at **03:37**, with a five-minute dispatch window, downtime consent and
resume-after-success.

- Saved: `03:35:49.388729643Z`.
- Slot: `2026-09-29T03:37:00Z`.
- Operation: `op-1fe3ebf43462b7d8c7ffaeef9103c96e`.
- Request: `schedule-f702d36eabb8f6a03aaed0252df8dea1-1790653020`.
- Created: `03:37:01.429410028Z`; finished: `03:37:24.871122981Z`, succeeded.
- Runtime workflow interval: `03:37:01.515394708Z`–`03:37:24.869639909Z`.
- Pack interval: `03:37:02.359044679Z`–`03:37:18.982626997Z`.
- Logical pier bytes: **1,041,195,020 → 631,021,580**. This is not a guaranteed
  reclaimed-space or loom-utilization measurement.
- Utility admission: 9,007,312,896 available RAM bytes; 434,808,745,984 disk bytes.

The script observed pack, start and complete stages; Stop was too brief to appear
in its polling samples. The completed run records `was_running:true`, explicit
stop/resume policy, and success. Runtime container replacement, fresh metrics,
running/healthy 1/1 state and direct `~zod` HTTP at port 18086 were verified.
Additional scheduler ticks produced no duplicate operation for the slot. Alpha,
panel-check and comet-check remained healthy; no incidents were reported.

The test policy was then **disabled**, revision **2**, at
`03:37:38.209158624Z`, with no next slot. No ongoing test routine remains enabled.
Private schema version 4 now requires 6S or newer for management. This accepts the
exercised timed fake-pier pack cycle; automatic comet maintenance remains gated.
