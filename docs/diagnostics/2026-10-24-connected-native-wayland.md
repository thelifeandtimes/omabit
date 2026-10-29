# Connected native Wayland lifecycle

The actual supported Omabit/Theme/DMs guarded set now combines the complete native
bar/keyboard panels, native shell API sync/pruning, a private software compositor,
real independent keyring sessions and synthetic Eyre subscriptions in one run.
Stable/dev/stable passes with15 positive subscriptions,10 deleted channels, six
unchanged session fingerprints and no new login. The single DM accepted without
an acknowledgement remains byte-identical and is never replayed on return.

Visible Theme UI has a valid palette and no operation/desktop error. DMs displays
the successful zero-conversation refresh result, with the intended unknown-action
banner still present. Actual popup mapping/navigation, Enter-control/Escape closure,
already-open popup closure on guard loss/drain, nine primary/widget constructions,
three native API unload observations and all13 API destruction records pass.
Twelve screenshots are retained; final DMs and Theme surfaces were visually checked.
The owned compositor exits zero and all guards exit.

The first combined run passed lifecycle preservation but visual inspection exposed
incomplete fixture reads: missing groups-ui heads and unsupported local palette
--file invocation. The synthetic handler now offers empty heads only when this
visual fixture explicitly opts in, and the private appearance adapter supports
only the exact private current-theme path. Initial colors.toml is now populated.
Trial2's assertion incorrectly demanded blank status after a successful refresh;
trial3 requires the actual successful empty-inbox status with no error. These are
fixture corrections, not changes to real network or desktop behavior.

The existing native original-guardian crash/recovery regression also passes after
those fixture corrections, with the same15 subscriptions/10 channels and preserved
sessions/uncertain DM. Original guardian receipts and source hashes verify; all
four scopes are inactive. No raw credentials/keyrings/account state are exported.

M299 supersedes M297/M298's separate connected-transport/compositor limitation for
clean native transitions. It does not combine shell SIGKILL/coordinator recovery
with the compositor in one run; that regression remains offscreen. It does not
qualify real groups message contents against this synthetic empty inbox, full
Omarchy shell, captured callbacks, Hyprland/IME, cross-root runtime ownership or a
public profile command. User ships, tmux, temp-ur, installed plugins and host desktop
remain untouched. Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-connected-native-wayland
