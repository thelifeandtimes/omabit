# Managed comet accepted — September 29 UTC

The collector explicitly accepted `development-managed` at the existing workstation
endpoint and runner ID `runner-f496feed1f7b08f31a5d82b94ac2f5fd`. Enrollment revision
advanced from 1 to 2; name, address and original enrollment timestamp were retained.
The desktop inventory was refreshed afterward.

`scripts/validate-managed-comet.py` then passed the approved fresh-identity gate.

| Action | Operation | Created UTC | Finished UTC |
| --- | --- | --- | --- |
| Register | `op-008be1a1413d19a80174b4a768dfaf0b` | 02:39:10.167656165 | 02:39:10.167656165 |
| First Start | `op-267de719ed8ec8f01db1a44fedbaf6ea` | 02:39:10.773557512 | 02:42:52.673892672 |
| Stop | `op-580fa0285b42ee73c98793a3785436aa` | 02:42:55.861948975 | 02:42:57.147244711 |
| Resume | `op-81970352b687355dc7fe7dedfe722423` | 02:42:58.647754429 | 02:43:05.217716531 |

Registration remained uninitialized/stopped with no invented ship name. First Start
generated **`~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod`**. Both starts
reached healthy 1/1 and direct loopback HTTP at port 18087 returned this same name.

Runtime container changed from
`6d392b6ef919c9665989fdc09aa5ca87538165798a16307e0cd0afe17458f092` to
`027c197670a1a7ca5cb9c4f89407ad27fd75d3f9d0938020634342763f5b6525`.
Fresh resource samples had a 4 GiB limit and ten tasks. Registration and operation
replay matched exactly. Login-code shape and absence from the snapshot were checked
without printing the code. An attempted pack was refused as unsupported for comets.

Final `comet-check` is running/healthy on `http://127.0.0.1:18087`. Alpha, panel-check
and import-check remained healthy; no incidents were reported. The database now
requires version-3-aware management (6R+) in `serve-development` mode. Host Ames
publication, peering and comet maintenance remain separate qualification work.
