#!/usr/bin/env bash
set -euo pipefail
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${root}"
source tools/env.sh
static() {
    command -v shellcheck >/dev/null 2>&1 || { echo "ERROR: shellcheck missing; see docs/development/setup.md" >&2; return 1; }
    python3 tools/check-repo.py
    node --check apps/console/assets/app.js
    node --check apps/console/assets/room-picker.js
    test -z "$(gofmt -l apps/console/*.go)"
    scratch="$(mktemp -d "${TMPDIR:-/tmp}/pocketlink-check.XXXXXX")"
    trap 'rm -rf -- "${scratch}"' EXIT
    cc -std=c11 -Wall -Wextra -Werror -Ifirmware/apps/board-check/main \
        tests/firmware/test_ui_pixel_math.c firmware/apps/board-check/main/ui_pixel_math.c \
        -o "${scratch}/test-ui"
    "${scratch}/test-ui"
    python3 tests/firmware/test_verify_firmware.py
    python3 tests/firmware/test_font_coverage.py
    for suite in config inbox dns portal_request pages activity; do
        cc -std=c11 -Wall -Wextra -Werror -Ifirmware/components/pocketlink_config \
            "tests/provisioning/test_${suite}.c" firmware/components/pocketlink_config/*.c \
            -o "${scratch}/test-${suite}"
        "${scratch}/test-${suite}"
    done
    cc -std=c11 -Wall -Wextra -Werror -Itests/ota/stubs -Ifirmware/components/pocketlink_ota -Ifirmware/components/pocketlink_config \
        tests/ota/test_boot.c firmware/components/pocketlink_ota/ota_boot.c -o "${scratch}/test-ota-boot"
    "${scratch}/test-ota-boot"
    node tests/provisioning/test_portal.cjs
    node tests/ota/test_console.cjs
    node tests/console/test_sessions.cjs
    (
        cd services/relay
        test -z "$(gofmt -l cmd internal)"
        go vet ./...
        go test -race ./...
        go build -trimpath -o "${scratch}/relay" ./cmd/server
    )
    python3 tools/smoke-server.py "${scratch}/relay"
    if command -v actionlint >/dev/null 2>&1; then actionlint .github/workflows/*.yml; else
        echo "ERROR: actionlint missing; install the version documented in docs/development/setup.md" >&2
        return 1
    fi
}
firmware() {
    command -v idf.py >/dev/null || { echo "Activate ESP-IDF 5.5.3 first" >&2; return 1; }
    [[ "$(idf.py --version)" == "ESP-IDF v5.5.3" ]] || { echo "ESP-IDF 5.5.3 required" >&2; return 1; }
    ./tools/test-ota-parser.sh
    for firmware_app in board-check pocketlink; do
    (
        scratch="$(mktemp -d "${TMPDIR:-/tmp}/pocketlink-firmware.XXXXXX")"
        trap 'rm -rf -- "${scratch}"' EXIT
        app="${root}/firmware/apps/${firmware_app}"
        cd "${app}"
        SDKCONFIG_DEFAULTS="${app}/sdkconfig.defaults" idf.py -B "${scratch}" \
            -D "SDKCONFIG=${scratch}/sdkconfig" build
        idf.py -B "${scratch}" merge-bin -o "${scratch}/FoloToy-AI-Passport-full.bin"
        python3 "${root}/tools/verify_firmware.py" "${scratch}"
        mkdir -p "${root}/dist/firmware/${firmware_app}"
        cp "${scratch}/FoloToy-AI-Passport-full.bin" "${root}/dist/firmware/${firmware_app}/"
        if [[ "${firmware_app}" == pocketlink ]]; then
            cp "${scratch}/FoloToy-AI-Passport.bin" "${root}/dist/firmware/${firmware_app}/pocketlink-ota.bin"
        fi
    )
    done
}
case "${1:---all}" in
    --static) static ;;
    --firmware) firmware ;;
    --all) static; firmware ;;
    *) echo "Usage: $0 [--static|--firmware|--all]" >&2; exit 2 ;;
esac
