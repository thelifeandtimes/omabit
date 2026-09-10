# Upstream references reviewed

Reviewed September 8, 2026. Upstream pages are mutable; this is a source review,
not a vendored dependency set or proof that the current registry image matches
GitHub `develop`. No upstream repository was cloned successfully in the isolated
build environment. Selected files and documentation were inspected via web.

## GroundSeg

- https://github.com/Native-Planet/GroundSeg
- https://manual.groundseg.app/guide/release-channels.html
  GroundSeg channel vocabulary: latest, edge, canary.
- https://manual.groundseg.app/guide/devmode.html
  Reference for attaching an existing tmux Dojo session in a container.
- https://github.com/Native-Planet/GroundSeg/blob/master/goseg/defaults/scripts.go
  Runtime wrapper and console operational patterns. This project's wrapper is
  original code using the same tmux approach, not a wholesale script copy.
- https://github.com/Native-Planet/GroundSeg/blob/master/goseg/defaults/defaults.go
  Version endpoint and default update configuration.
- https://github.com/Native-Planet/GroundSeg/blob/master/goseg/defaults/version.go
  Manifest structure: groundseg → channel → vere; repo, tag, architecture digest.
- https://github.com/Native-Planet/GroundSeg/blob/master/goseg/config/version.go
  Release/image resolution reference.
- https://github.com/Native-Planet/GroundSeg/blob/master/goseg/docker/urbit.go
  Container lifecycle and image selection reference.
- https://github.com/Native-Planet/GroundSeg/blob/master/goseg/handler/urbit.go
  Broader feature inventory; not every action belongs in this milestone.

## Vere / Urbit

- https://docs.urbit.org/user-manual/running/vere
  Keyed/comet/fake bootstrap flags, loom and pill options; distinct Vere release
  paces live, soon, edge.
- https://github.com/urbit/vere/tree/develop/docker
- https://github.com/urbit/vere/blob/develop/docker/get_urbit_code.sh
  Existing HTTP loopback payload for +code. Our adapter reads the target pier's
  actual loopback port rather than assuming 12321 in a shared namespace.
- https://github.com/urbit/vere/blob/develop/pkg/vere/io/http.c
  `_http_write_ports_file` writes `<port> insecure|secure loopback|public` into
  `.http.ports`; multiple loopback listeners can select different available ports.
- https://github.com/urbit/vere/blob/develop/pkg/vere/io/ames.c
  `_ames_czar_port` and `_ames_czar_lane`: fake-galaxy loopback routing and
  the 31337 + galaxy-index port convention.

## Later integration references

- https://github.com/nisfeb/urbit-boot-automation
- https://github.com/nisfeb/urbtop
- https://github.com/Native-Planet/Anchor
- https://github.com/omacom/omarchy/tree/quattro

These are design references, not dependencies installed by this artifact. The
pilot contains no urbtop polling, QML widget or Anchor modification. The canonical
Urbit galaxy suffix table is protocol data used to select fake galaxy indices.

## General operational references

- https://specifications.freedesktop.org/basedir/latest/
- https://docs.docker.com/engine/install/linux-postinstall/
- https://docs.docker.com/engine/network/port-publishing/
- https://docs.docker.com/engine/network/drivers/bridge/
- https://www.rfc-editor.org/rfc/rfc6265
- https://man.openbsd.org/sshd
- https://man.openbsd.org/ssh

The source and tests remain the definition of what this pilot actually does.
