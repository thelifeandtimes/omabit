# Guided import layout fixtures

The public guided-preview path passed nine cases using temporary trees and a
fixture identity detector: arbitrary source folder, planet with repeated identity,
known comet, legacy `pier` child, custom-root legacy child, custom-root named
in-place enrollment, directory collision, registered-name collision, and a source
category that disagrees with detected network. The plan follows detected identity;
the basename is not used as the ship's identity. Preview preserves the source
fingerprint and does not register or publish a destination.

Existing runtime-engine fixtures passed same-filesystem rename, injected
cross-device copy, source replacement refusal, interrupted-copy fencing, custom
mount/disk admission, and guided in-place recovery. The named/legacy import fixture
now explicitly checks that initialized runtime commands use `/urbit/zod` and bind
only the exclusive instance parent. Import itself does not start a runtime.

The focused race run passed both packages. The [evidence manifest](2026-10-22-import-layout.json)
records the exact command, test names, source hashes and retained log. These are
meaningful fixture checks of the existing layout behavior; no production import
implementation or path migration was introduced in this slice.

This is not a manual GUI or real Docker/Vere import qualification. Cross-device
copy is selected by injected `EXDEV`; no host filesystem was mounted. A fresh
disposable engine/VM and isolated desktop are still required for the guided import,
progress/recovery, explicit Start and identity-resume gate. The excluded reported
pier, managed `~lismyl-lismyl`, tmux ships and `~/temp-ur` were not inspected or used.
