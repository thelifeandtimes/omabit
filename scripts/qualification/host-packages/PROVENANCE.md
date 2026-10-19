# Source provenance and qualification boundary

The reusable source is a sanitized promotion, not a copy of generated VM state.
Its native cases originate in the following **captured script snapshots** from the
actual 2026-10-19 qualification runs. Hashes below identify those historical files,
not the promoted code in this directory.

| Historical script | SHA-256 |
|---|---|
| `run-fixed-20261019T023329Z/harness/fixed_suite.py` | `02f3c823769ab13640470ca6a78a49e398fa0dd93f1b9262f10fe00824e20b60` |
| `run-fixed-20261019T023329Z/harness/guest.py` | `db4d0cbe07ca8b9a26c6dc89699a8137dbf6ac8a560e71e8ef37abda6e4a8021` |
| `run-fixed-20261019T023329Z/harness/pacman_guest.py` | `c1e5bcbfa39594ead00fda98b2305365da7c196e5ab2d05b307e6095d2e08f67` |
| `run-colon-20261019T033848Z/harness/colon_guest.py` | `2e3fbbd2f291b8b2016e9c2cfb96268e0b5970a6b02162e68a79de1c36f39365` |

Local capture paths are relative to `build/hostcompat-native/`. Those files and
their raw evidence remain ignored; running the reusable driver needs none of them.
The corresponding exact runner/core manifest hashes and native tool versions are
recorded in [README.md](README.md).

## Promotion map

* `suite.py`: full-run assertion groups from `fixed_suite.py` and the focused
  native APT colon authority case from `colon_guest.py`.
* `fixtures.py`: signed .deb repository/package construction from `guest.py` and
  Arch package/repository/trust setup from `pacman_guest.py`; prerequisite install
  and image download are left to independent guest provisioning.
* `guards.py` and `qualify.py`: new mandatory read-only admission, version/hash
  pins, inert help/plan/validate and explicit mutation dispatch.
* `test_guards.py`: new unprivileged guard and CLI tests. No native authentication
  result is simulated or asserted as qualification by these tests.

The promotion parameterizes the staging paths and native `_apt` identity, starts
from fresh fixtures, selects APT/Pacman groups explicitly and retains public
multisignature negative inputs. Test commands still invoke real native tools.
All previous acceptance/denial assertions remain native-case assertions; the
retained read-only CLI always reports engine qualification as `unqualified`.

No complete native matrix was rerun for this promotion. The captured full native
pass belongs to the historical snapshot hashes above. The stronger guards and
reorganized fresh setup have CI-safe source coverage, not a new native-platform
qualification claim. Future runs should retain the reusable `.py` file hashes,
candidate hash/schema/version, provisioned VM identity and actual native outputs.

No VM images, runner binaries, package archives, generated private keys or raw
logs are committed with this source. Fixture signing keys are generated only after
disposable-guest admission and remain in private directories inside that guest.
