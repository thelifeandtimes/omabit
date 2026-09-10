# Applications

Each directory under `apps/` is an independently scoped Omabit workstream. An
application should own its commands, tests, packaging, and operational docs.

- `urbit-host/`: imported Go ship-management CLI and host service.
- `pals/`: discovery integration brief; implementation not started.
- `tlon-messenger/`: Omarchy-native messenger brief; implementation not started.

Tend is temporarily at the repository root while active work uses its existing
paths. See `../docs/REPOSITORY.md` before creating `apps/tend/`.
