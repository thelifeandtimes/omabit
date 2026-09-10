# Omabit Workstreams

## Product thesis

Omabit gives Omarchy users a native path to user-owned identity, hosting,
discovery, communication, and multiplayer applications. Urbit's `@p` identity
and ship-to-ship network fill capabilities that Linux desktops do not normally
provide as a coherent platform; Omarchy plugins make those capabilities feel
like part of the desktop rather than a separate web stack.

The project should not require an Omabit-operated identity provider or central
application database. Some Urbit networking and optional third-party hosting
infrastructure may still exist, but users retain control of their identities,
ships, and application state.

## 1. Urbit host manager

Location: `apps/urbit-host/`

The host manager is the operational foundation. It manages disposable and,
after validation, valuable Urbit ships on an Omarchy workstation or remote Linux
host. It remains an independent Go CLI/service named `urbitctl` for now.

Its responsibilities include lifecycle, host inventory, SSH transport,
container/runtime control, resource observations, and safe archival. It does not
own Tend, Pals, or Tlon application data and must not become a mandatory central
service for their peer-to-peer behavior.

## 2. Tend

Current location: repository root, pending coordinated move to `apps/tend/`

Tend is a polished Apple Reminders-class demonstration of building a native,
multiplayer Omarchy application on Urbit. It also solves a practical problem:
shared personal task data without Apple ID, iCloud, or a replacement centralized
Omabit account service.

The `%tend` Gall agent owns durable state and collaboration authority. The
Omarchy plugin provides the native desktop UI and connects to the user's ship
through Eyre. Tend may consume identity and connection configuration exposed by
shared contracts, but it must remain usable independently of the messenger UI.

## 3. Pals discovery

Location: `apps/pals/`

The Pals workstream will help new Omarchy users discover other participating
users and understand who is reachable through Urbit identity. It is a flagship
onboarding and network-discovery feature, not merely a Tend address picker.

Before implementation, specify the existing `%pals` contract, consent and
visibility rules, trust/spam controls, profile fields, caching behavior, and the
boundary between discovery data and application-specific contact lists.

## 4. Tlon Messenger

Location: `apps/tlon-messenger/`

The Tlon Messenger workstream will provide an Omarchy-native messaging client
that uses the user's Urbit identity and existing Tlon-compatible data rather
than creating a second messaging network. Alongside Pals, it is a flagship way
for a new Omarchy user to meet and communicate with other users.

Before implementation, choose the supported Tlon/Urbit protocol surface,
authentication and ship-connection model, notification behavior, offline cache,
deep-link contracts, and whether the first release is a focused client or a
broad Groups-compatible replacement.

## Shared contracts to design deliberately

- Ship connection profiles and secure desktop authentication.
- Canonical `@p` display, avatar, and presence primitives.
- Deep links among discovery, messaging, and collaborative apps.
- Desktop notifications and background service ownership.
- Installation and update behavior for Omarchy plugins and Urbit desks.
- Versioned APIs that prevent one application from reading another's internal
  database or repository files directly.
