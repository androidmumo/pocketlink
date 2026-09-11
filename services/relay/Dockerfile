# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.26.8-bookworm AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src/services/relay
COPY apps/console/ /src/apps/console/
COPY services/relay/go.mod services/relay/go.sum ./
RUN go mod download
COPY services/relay/ ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/relay ./cmd/server \
    && mkdir -p /out/data

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/androidmumo/pocketlink"
LABEL org.opencontainers.image.title="PocketLink relay foundation"
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/relay /relay
COPY --from=build --chown=65532:65532 /out/data /data
USER 65532:65532
ENV POCKETLINK_LISTEN=0.0.0.0:8080 \
    POCKETLINK_DATABASE=/data/relay.db \
    GOMEMLIMIT=128MiB
EXPOSE 8080/tcp
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/relay", "healthcheck"]
ENTRYPOINT ["/relay"]
