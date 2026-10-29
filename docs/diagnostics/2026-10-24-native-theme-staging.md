# Native headless theme staging and retained appearance

The optional --native-theme-staging lifecycle mode copies and hashes the actual
Omarchy theme setter, template renderer, palette resolver and built-in templates.
Those copies run only inside the existing private mount/PID/network namespace,
with private HOME and native headless mode. Native staging uses the real lock,
next-theme directory, templates, swap and temporary theme name. Desktop/compositor
controls remain inert; wallpaper and app-retint fanout are excluded.

The actual Theme/DMs service and offline CLI scenario passes with this native file
staging: stable checkpoint, partial candidate failure, exact helper/channel drain,
hub/keyring shutdown, failed restore, startup refusal, verified retry, completed
read-only replay, forced helper exit after a native staging effect and another
verified retry. All final theme files match the frozen snapshot byte-for-byte.
Copied native inputs and applied harness/plugin source hashes are independently
checked; both trial scopes are inactive. Native staging preserved existing frozen
files instead of replacing them with template output on this source version.
The applied mode requires --connected --appearance-offline --native-theme-staging;
invalid combinations and host entry refuse before runtime work.

This supersedes M262's wholly simulated theme-filesystem boundary for headless
staging/templates only. Native app-retint, font-triggered shell restart, background,
actual compositor, complete supported-plugin quiescence and durable collection
activation remain open. No native command ran against the host desktop or ships.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-theme-staging`.
