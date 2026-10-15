# Omabit 0.1.0-rc.4 — onboarding feedback

Linux/AMD64 prerelease, schema16 and the RC.3 owner-user runtime.

- Fresh collectors show the Urbit introduction and **Start Setup →**, without a
  misleading inventory/SSH error. **Check again** follows an explicit check/setup.
- First-run setup saves **Always-on machine** / **Laptop / sometimes off**. Laptop
  fleets offer a remote-server setup guide and connection entry point.
- Comet names are required inputs, with incrementing `comet-n` placeholders. Owned
  identities derive their names from @p. Port suggestions check the selected host's
  actual listeners and reserved ports through a new operator-only read-only RPC.
- Default and named-root destinations are generated. The pier-path editor appears
  only after **Custom path** is selected, retaining the existing storage convention.
- HTTP help explains guest 8080 versus the selected host port.
- Confirmed new comet/fake/key creation submits one first Start after verified
  registration, binds the returned revision, and opens the ship's management view.
  Uncertain registration/boot responses are not automatically retried.
- Browser/+code actions require a fresh healthy, revision-current runtime with no
  pending identity verification, recovery or active operation.
- Monitor actions use a native Qt window with an explicit top-level relationship;
  opening, reusing, focusing across workspaces, closing/reopening and ship selection
  are exercised. The fix also mapped successfully in the real sourcebox shell.
- Removed the development networking paragraph from compact fleet cards.

Install as your ordinary Omarchy user:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

The standalone runner/collector pins must advance together for host port checking.
The runtime image, schema, resource limits and frozen draft Recommended Apps pill
are unchanged. The draft's application/OTA limitations remain. This release adds
no remote automatic provisioning; the remote setup view guides host installation
and enrollment. Existing backend CLI registration commands remain explicit and
stopped; automatic first boot is the confirmed GUI workflow.
