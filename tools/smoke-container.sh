#!/usr/bin/env bash
set -euo pipefail
image="${1:?Pass an image name}"
platform="${2:-linux/amd64}"
container=""
cleanup() {
    if [[ -n "${container}" ]]; then
        docker logs "${container}" || true
        docker rm -f -v "${container}" >/dev/null || true
    fi
}
trap cleanup EXIT
container="$(docker run -d --platform "${platform}" --read-only --cap-drop ALL \
    --security-opt no-new-privileges:true --pids-limit 128 --memory 192m \
    --mount type=volume,destination=/data \
    --tmpfs /tmp:size=8m,mode=1777 "${image}")"
ready=false
for attempt in {1..30}; do
    if docker exec "${container}" /relay healthcheck; then ready=true; break; fi
    if [[ "$(docker inspect -f '{{.State.Running}}' "${container}")" != true ]]; then break; fi
    sleep 1
done
[[ "${ready}" == true ]] || { echo "Container never became ready" >&2; exit 1; }
docker stop --time 15 "${container}" >/dev/null
[[ "$(docker inspect -f '{{.State.ExitCode}}' "${container}")" == 0 ]]
# Reopen the same volume to verify persistence and migration idempotency.
docker start "${container}" >/dev/null
ready=false
for attempt in {1..30}; do
    if docker exec "${container}" /relay healthcheck; then ready=true; break; fi
    sleep 1
done
[[ "${ready}" == true ]]
echo "Container start/reopen/shutdown (${platform}): PASS"
