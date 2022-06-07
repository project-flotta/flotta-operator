# Build the flotta-edge-api binary
FROM golang:1.17 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

COPY vendor/ vendor/

# Copy the go source
COPY cmd/httpapi/main.go main.go
COPY api/ api/
COPY internal/common internal/common
COPY internal/edgeapi internal/edgeapi
COPY models/ models/
COPY pkg/ pkg/
COPY restapi/ restapi/
COPY backend/ backend/
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -a -o flotta-edge-api main.go

# Use distroless as minimal base image to package the flotta-edge-api binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/flotta-edge-api .
USER 65532:65532

ENTRYPOINT ["/flotta-edge-api"]
