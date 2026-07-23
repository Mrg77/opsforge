# opsforge — distroless container image.
#
# Multi-stage build: a Go stage compiles the fully static (CGO_ENABLED=0)
# binary, and a distroless/static:nonroot final stage ships just that binary.
# opsforge without Homebrew uses its GitHub-releases install backend, so the
# image needs no package manager — just the static binary.
#
# Build (single-arch):
#   docker build --build-arg VERSION=v1.2.3 -t ghcr.io/mrg77/opsforge:v1.2.3 .
#
# Build (multi-arch, requires buildx):
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     --build-arg VERSION=v1.2.3 -t ghcr.io/mrg77/opsforge:v1.2.3 --push .
#
# Run:
#   docker run --rm ghcr.io/mrg77/opsforge:v1.2.3 audit --json

# --- build stage ----------------------------------------------------------
FROM golang:1.26 AS build

WORKDIR /src

# Cache module downloads independently of the source tree.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# VERSION is injected into the same var GoReleaser stamps
# (github.com/Mrg77/opsforge/cmd.version), so `opsforge --version` is correct.
# TARGETOS/TARGETARCH are provided by buildx for multi-arch builds; they
# default to the build host otherwise.
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build \
      -trimpath \
      -ldflags "-s -w -X github.com/Mrg77/opsforge/cmd.version=${VERSION}" \
      -o /opsforge .

# --- final stage ----------------------------------------------------------
FROM gcr.io/distroless/static:nonroot

# OCI image metadata (standard org.opencontainers.image.* keys).
LABEL org.opencontainers.image.title="opsforge" \
      org.opencontainers.image.description="Forge your DevOps workstation: pick your CLIs, get a fully wired shell, and enforce policy-as-code guards." \
      org.opencontainers.image.source="https://github.com/Mrg77/opsforge" \
      org.opencontainers.image.url="https://github.com/Mrg77/opsforge" \
      org.opencontainers.image.documentation="https://github.com/Mrg77/opsforge#readme" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="Mrg77"

COPY --from=build /opsforge /opsforge

# distroless/static:nonroot already runs as an unprivileged user (uid 65532).
USER nonroot:nonroot

ENTRYPOINT ["/opsforge"]
