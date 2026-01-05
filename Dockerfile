# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o binarylane-cloud-controller-manager ./cmd/binarylane-cloud-controller-manager

# Final stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /

COPY --from=builder /workspace/binarylane-cloud-controller-manager .

USER 65534:65534

ENTRYPOINT ["/binarylane-cloud-controller-manager"]
