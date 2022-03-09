@echo off

del qljs.msix
rem cd C:\Users\strager\Documents\Projects\quick-lint-js\myappx
makeappx pack /hashAlgorithm SHA256 /v /d sauce /p qljs.msix
signtool sign /fd SHA256 /a /f "C:\Users\strager\Documents\Projects\quick-lint-js\WapProjTemplate2\WapProjTemplate2_TemporaryKey.pfx" /p hello qljs.msix

rem @@@ TODO (next steps):
rem 1. try jsign for .exe signing
rem 2. if jsign works, use it instead of osslsigncode
rem 3. add appx support to jsign (preferred) or osslsigncode
rem    (if jsign doesn't work)

rem CI build:
rem * makeappx to create .msix

rem release:
rem * sign .msix
rem   * signing on Windows during release sucks
rem     * possible solution: fb-util-for-appx
rem     * possible solution: osslsigncode
rem       * I could write it
rem       * problem: licensing/copyright due to FB ties
rem     * possible solution: https://github.com/microsoft/msix-packaging/blob/johnmcpms/signing/src/msix/pack/Signing.cpp
rem       * might be windows-only
rem       * backstory: https://github.com/mtrojnar/osslsigncode/issues/51
rem     * possible solution: https://github.com/ebourg/jsign
rem       * I could write it: https://github.com/ebourg/jsign/issues/81
rem       * problem: licensing/copyright due to FB ties
rem     * possible solution: write my own signtool
rem       * problem: licensing/copyright
rem       * problem: OpenSSL sucks; Go crypto might suck too
rem * generate winget .installer.yaml


rem * Windows installer -- signtool (Windows only)
rem * Windows .exe      -- osslsigncode (*)
rem * Linux .exe        -- gnupg (*)
rem * macOS .exe        -- codesign (macOS only)

rem winget install -m manifests\q\quick-lint\quick-lint-js\1.0.2.0
rem winget hash --msix ..\..\myappx\qljs.msix
rem winget validate manifests\q\quick-lint\quick-lint-js\1.0.2.0

rem HUGE caveat: when testing, you can't change the data
rem behind an install URL. You must create a new URL.
rem (Changing the hashes isn't enough!)

rem caveat: .msix must be signed in order to install with
rem winget.

rem after changing the .msix:
rem * get .msix hashes and put in .yaml file: winget hash --msix ..\..\myappx\qljs.msix
rem * change installer URL
rem * uninstall old version
rem * install new version: winget install -m manifests\q\quick-lint\quick-lint-js\1.0.2.0
rem * don't need to change version number.
