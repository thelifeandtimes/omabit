---
name: omabit-plugin-developer
description: Build or adapt standalone Omarchy interfaces for Urbit Gall or Grubbery applications, including direct Eyre transport, optional Omabit authentication, backend readiness and isolated development profiles.
---

# Develop an Omarchy–Urbit application

Deliver **“Multiplayer apps without a subscription.”** Start with a useful action
whose state belongs on the user's ship and can be shared with another device or
person. Omabit manages ships and assists authentication; the plugin owns its UI,
backend readiness and direct ship traffic. Discovery belongs in Omart. Keep the
plugin usable without Omabit, including with a hosted ship.

Read the target repository's contributor instructions and nearest application
documentation. Inspect its existing manifest, transport, storage and UI conventions
before changing them. Read only the reference needed for the current slice:

- [Shared visual and interaction conventions](references/interface-style.md)
- [Interface, sign-in and backend readiness](references/application.md)
- [Development isolation and acceptance](references/development.md)

Work in small vertical slices:

1. Establish the actual backend contract: desk/nexus and publisher, Gall agent or
   Grubbery API, readiness/permission probes, wire shapes and supported versions.
   A desk label alone does not establish an API or a running agent.
2. Make standalone URL/+code sign-in work, with direct authenticated reads and
   explicit writes. Keep login codes out of argv, logs and saved preferences.
3. Give the plugin an independent Omabit approval receiver when integrating that
   feature. Use the implemented handoff contract and installed-helper declaration;
   do not invent a shared cookie file, credential RPC or Omabit data proxy.
4. Present missing-backend installation in this plugin, naming the exact backend
   and publisher. Model installation, readiness and permission approval separately.
5. Test the real UI and transport in a disposable ship/desktop boundary. Record
   what was exercised and what still uses an inert adapter before widening scope.

Theme and DMs are the initial authentication integration pair. Both use `%groups`
backends, but still probe the agent/API and version that the particular feature
needs. Tend is the later full multiplayer and missing-backend demonstration.
Consult Talon's reconnect policy and Lattice's Gall/Grubbery client patterns when
working on their respective concerns; inspect current source, not remembered APIs.

The source Omabit documents `docs/application-connections.md` and
`docs/application-handoff-v1.md` define the versioned approval implementation.
They describe durable attempts, exact installed artifacts, independent sessions
and cleanup. Follow that contract when available; standalone sign-in does not
depend on it. A future API or a roadmap paragraph is not an implemented capability.

Use the task's existing authorization. A button or acknowledgement can express the
required effect without asking for the same approval again in chat. Stop only for
genuinely missing authority or an unresolved target. Never broaden a failed request
to another ship, publisher, credential, transport or profile without that authority.

Respect the user's excluded ships, directories and installed plugins without
probing them. For this platform work, production services and ships remain outside
experiments. Make every experimental instance visibly distinct, and isolate its
state, credentials, IPC and side effects as well as its display label.
