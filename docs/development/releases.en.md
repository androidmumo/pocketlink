[简体中文](releases.md) | English

# Firmware release maintenance

User downloads belong on [GitHub Releases](https://github.com/androidmumo/pocketlink/releases). Each release has four files: `pocketlink-full.bin` (USB at 0x0), `pocketlink-ota.bin`, `manifest.json`, and `SHA256SUMS`. Source archives and expiring CI artifacts are not the user download channel.

## Current package

The original 0.5.0 files live in `firmware/releases/0.5.0/`. Their source is `78a5348`, recorded in `source.txt`; the signed OTA matches production without re-signing. Tag `firmware/pocketlink/v0.5.0` points to the commit containing publication configuration. The release body separately identifies the actual firmware source commit.

## Subsequent firmware

In Actions, run **Signed firmware package** on main with a new version and strictly increasing sequence. It builds both images, runs host checks, signs with the existing Secret and publishes a Release. It does not deploy Docker, change a server OTA channel or automatically install on devices. Administrators still upload the signed package to their server and publish it there.

Do not run signing for documentation-only changes. Never reuse a version/sequence for different firmware or overwrite published assets. Release attachments take precedence over the historical 0.5.0 repository snapshot; subsequent build outputs are not written back to main.

To preserve an already verified and signed bundle byte-for-byte, place its four files and `source.txt` in `firmware/releases/<version>/`, run `python3 tools/check-release.py firmware/releases/<version>`, commit and push tag `firmware/pocketlink/v<version>`. **Publish verified firmware bundle** handles publication. This path is used for 0.5.0.

## Permissions and failures

With explicit user authorization, only the two publishing jobs receive `contents: write`; default workflow permissions remain read-only. Checks cover signature, SHA256, full-image/OTA equality, protected partitions and tag/version equality. The release starts as a draft and becomes public only after all four uploads succeed. Existing releases are not overwritten. Failed uploads may leave a draft; inspect the failure and missing files before explicitly resolving that draft, rather than blindly retrying or deleting a published release.

The current release title and body say Development so users can find the latest downloadable package. Physical acceptance limits are documented per release. Publication does not imply successful installation on a device.
