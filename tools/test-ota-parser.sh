#!/usr/bin/env bash
# Compile the actual device parser with the same IDF mbedTLS/cJSON sources.
set -euo pipefail
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
: "${IDF_PATH:?Activate ESP-IDF 5.5.3}"
scratch="$(mktemp -d "${TMPDIR:-/tmp}/pocketlink-parser.XXXXXX")"
trap 'rm -rf -- "$scratch"' EXIT
cmake -S "$IDF_PATH/components/mbedtls/mbedtls" -B "$scratch/build" -DENABLE_PROGRAMS=OFF -DENABLE_TESTING=OFF >"$scratch/build.log" 2>&1
cmake --build "$scratch/build" --target mbedcrypto -j 4 >>"$scratch/build.log" 2>&1 || { cat "$scratch/build.log"; exit 1; }
# Test-only ephemeral key; never persist or print private material.
umask 077
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -pkeyopt ec_param_enc:named_curve -out "$scratch/key" 2>/dev/null
openssl pkey -in "$scratch/key" -pubout -out "$scratch/public" 2>/dev/null
python3 - "$scratch" <<'PY'
import base64,json,subprocess,sys
from pathlib import Path
p=Path(sys.argv[1])
payload=json.dumps(dict(format=1,board='ai-passport-esp32c3',app='pocketlink',version='test-1',sequence=7,size=288,sha256='a'*64),separators=(',',':')).encode()
(p/'payload').write_bytes(payload)
subprocess.run(['openssl','dgst','-sha256','-sign',str(p/'key'),'-out',str(p/'signature'),str(p/'payload')],check=True)
(p/'manifest').write_text(json.dumps(dict(payload=base64.b64encode(payload).decode(),signature=base64.b64encode((p/'signature').read_bytes()).decode())))
(p/'trust.c').write_text('const char trust_start[] __asm__("_binary_trust_pem_start") = '+json.dumps((p/'public').read_text())+';\n')
PY
cc -std=c11 -Wall -Wextra -Werror -I"$root/tests/ota/stubs" \
 -I"$root/firmware/components/pocketlink_ota" -I"$root/firmware/components/pocketlink_config" \
 -I"$IDF_PATH/components/json/cJSON" -I"$IDF_PATH/components/mbedtls/mbedtls/include" \
 "$root/tests/ota/test_manifest.c" "$root/firmware/components/pocketlink_ota/ota_manifest.c" \
 "$root/firmware/components/pocketlink_config/pocketlink_config.c" \
 "$IDF_PATH/components/json/cJSON/cJSON.c" "$scratch/trust.c" \
 "$scratch/build/library/libmbedcrypto.a" -lm -o "$scratch/test-parser"
"$scratch/test-parser" "$scratch/manifest"
