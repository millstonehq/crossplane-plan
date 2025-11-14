VERSION 0.8
PROJECT millstonehq/crossplane-plan

IMPORT ../../lib/build-config/go AS go
IMPORT ../../lib/build-config/base AS base

# deps downloads and caches Go dependencies
deps:
    FROM go+base-go --GOLANG_VERSION=1.25
    WORKDIR /app

    COPY go.mod go.sum ./
    RUN go mod download

    SAVE ARTIFACT go.mod
    SAVE ARTIFACT go.sum

# build compiles the crossplane-plan binary
build:
    FROM +deps

    COPY --dir cmd pkg ./
    COPY go.mod go.sum ./

    # Build for target architecture with CGO disabled for static binary
    # TARGETARCH is built-in and set automatically by Earthly based on --platform
    ARG TARGETARCH
    RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
        -ldflags="-w -s" \
        -o /app/bin/crossplane-plan \
        ./cmd/crossplane-plan

    SAVE ARTIFACT /app/bin/crossplane-plan AS LOCAL bin/crossplane-plan

# test runs unit tests with coverage
test:
    FROM +deps
    
    COPY --dir cmd pkg ./
    COPY go.mod go.sum ./
    
    # Run tests with coverage
    RUN CGO_ENABLED=0 go test -v -cover ./...

# lint runs go vet and other linting
lint:
    FROM +deps
    
    COPY --dir cmd pkg ./
    COPY go.mod go.sum ./
    
    RUN go vet ./...
    RUN go fmt ./...

# image builds the container image
image:
    # Multi-platform build - TARGETPLATFORM/TARGETARCH are built-in and set by Earthly
    ARG TARGETPLATFORM
    ARG TARGETARCH
    FROM --platform=$TARGETPLATFORM go+base-go-runtime

    USER nonroot
    WORKDIR /app

    # Copy binary from build stage
    COPY +build/crossplane-plan /app/crossplane-plan

    ENTRYPOINT ["/app/crossplane-plan"]

    # Save image (--push only activates when running earthly --push +publish)
    ARG tag=latest
    SAVE IMAGE --push ghcr.io/millstonehq/crossplane-plan:${tag}

# publish pushes multi-arch images to ghcr.io
publish:
    FROM alpine:latest
    ARG tag=latest

    # Build and push both amd64 and arm64 images
    # Run with: earthly --push +publish
    # Authenticate first: docker login ghcr.io
    BUILD --platform=linux/amd64 --platform=linux/arm64 +image --tag=$tag

# kubedock-deps clones kubedock repo and downloads dependencies
kubedock-deps:
    FROM go+base-go --GOLANG_VERSION=1.25
    WORKDIR /kubedock

    # Clone kubedock at specific version
    ARG KUBEDOCK_VERSION=0.18.3
    RUN git clone --depth 1 --branch ${KUBEDOCK_VERSION} https://github.com/joyrex2001/kubedock.git .

    RUN go mod download

    SAVE ARTIFACT go.mod
    SAVE ARTIFACT go.sum

# kubedock-build compiles kubedock binary for multi-arch
kubedock-build:
    FROM +kubedock-deps

    # Build for target architecture
    ARG TARGETARCH
    RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
        -ldflags="-w -s" \
        -o /app/bin/kubedock \
        .

    SAVE ARTIFACT /app/bin/kubedock

# kubedock-image builds the kubedock container image
kubedock-image:
    ARG TARGETPLATFORM
    ARG TARGETARCH
    FROM --platform=$TARGETPLATFORM go+base-go-runtime

    USER nonroot
    WORKDIR /app

    # Copy binary from build stage
    COPY +kubedock-build/kubedock /usr/local/bin/kubedock

    ENTRYPOINT ["/usr/local/bin/kubedock"]

    # Save image
    ARG tag=0.18.3
    SAVE IMAGE --push ghcr.io/millstonehq/kubedock:${tag}

# kubedock-publish pushes multi-arch kubedock images to ghcr.io
kubedock-publish:
    FROM alpine:latest
    ARG tag=0.18.3

    # Build and push both amd64 and arm64 images
    # Run with: earthly --push +kubedock-publish
    BUILD --platform=linux/amd64 --platform=linux/arm64 +kubedock-image --tag=$tag

# all runs all checks and builds
all:
    BUILD +test
    BUILD +lint
    BUILD +build
