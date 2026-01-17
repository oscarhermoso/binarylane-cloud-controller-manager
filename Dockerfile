# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /workspace

# Copy go mod files (cached layer - only invalidated if deps change)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY cmd/ cmd/
COPY internal/ internal/

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

# Build with cache mounts for faster incremental builds
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -a -mod=readonly -ldflags="-w -s -X main.version=${VERSION}" \
    -o binarylane-cloud-controller-manager ./cmd/binarylane-cloud-controller-manager

FROM alpine:3.23.2
RUN apk add --update --no-cache ca-certificates

WORKDIR /

COPY --from=builder /workspace/binarylane-cloud-controller-manager .

USER 65534:65534

ENTRYPOINT ["/binarylane-cloud-controller-manager"]
