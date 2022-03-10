# quick-lint-js Scoop manifest

This directory contains manifests for [Scoop][], a Windows
package manager.

## Publishing

1. Every build, [CI
   uploads](../../.github/workflows/build-static.yml)
   `quick-lint-js.json` as-is to
   https://c.quick-lint-js.com/builds/COMMIT_HASH/scoop/quick-lint-js.json.
2. When [making a release](../../docs/RELEASE.md), the
   release engineer copies
   https://c.quick-lint-js.com/builds/COMMIT_HASH/scoop/quick-lint-js.json
   as-is to
   https://c.quick-lint-js.com/releases/VERSION/scoop/quick-lint-js.json.
3. When [making a release](../../docs/RELEASE.md), for each
   released version, the
   [sync-releases-to-scoop script](./sync-releases-to-scoop)
   reads
   https://c.quick-lint-js.com/releases/VERSION/scoop/quick-lint-js.json,
   hashes the referenced .zip files, adds the hashes to the
   manifest, and uploads the manifest with hashes to
   https://c.quick-lint-js.com/scoop/VERSION/quick-lint-js.json.
   (https://c.quick-lint-js.com/releases/VERSION/scoop/quick-lint-js.json
   remains unchanged.)

@@@ ideas

***
CI
  * copy manifest
release
  * instructions: update version numbers
  * Scoop publish script: generate manifests with hashes for all versions (like Debian)
+ no conflict/confusion with signing (e.g. SHA256SUMS)
+ similar to other package managers (Debian)
+ Scoop publish script works with all versions, not just latest
- confusing manifest in CI
- new step in release process
***

CI
  * (nothing)
release
  * instructions: update version numbers
  * Scoop publish script: generate manifest with hashes (like Debian)
+ no confusing manifest in CI
+ no conflict/confusion with signing (e.g. SHA256SUMS)
+ similar to other package managers (Debian)
- new step in release process
- Scoop publish script works only for latest release, not all (unlike Debian)

CI
  * (nothing)
release
  * instructions: update version numbers
  * new script: generate manifest with hashes
+ no confusing manifest in CI
- new step in release process
- scoop manifest isn't hashed into SHA256SUMS

CI
  * copy manifest
release
  * instructions: update version numbers
  * sign script: update manifest with hashes
+ fewer release process steps
- confusing manifest in CI
- sign script doesn't just sign anymore

CI
  * (nothing)
release
  * instructions: update version numbers
  * sign script: generate manifest with hashes
+ fewer release process steps
- sign script reads from source repo
- sign script doesn't just sign anymore

CI
  * copy manifest
release
  * instructions: update version numbers
+ fewer release process steps
- confusing manifest in CI
- no hashes in manifest
- autoupdate breaks (https://github.com/ScoopInstaller/Scoop/issues/4797)

[Scoop]: https://scoop.sh/
