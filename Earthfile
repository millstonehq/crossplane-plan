VERSION 0.8
PROJECT millstonehq/crossplane-plan

IMPORT ../../lib/build-config/go AS go
IMPORT ../../lib/build-config/base AS base

# deps downloads and caches Go dependencies
deps:
    FROM go+base-go --GOLANG_VERSION=1.22
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
    
    # Build for linux/amd64 with CGO disabled for static binary
    RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="-w -s" \
        -o /app/bin/crossplane-plan \
        ./cmd/crossplane-plan
    
    SAVE ARTIFACT /app/bin/crossplane-plan AS LOCAL bin/crossplane-plan

# test runs unit tests with coverage gate
test:
    FROM +deps
    
    COPY --dir cmd pkg scripts ./
    COPY go.mod go.sum .coverageignore ./
    
    # Run tests
    RUN CGO_ENABLED=0 go test -v ./...
    
    # Run coverage gate (business logic only)
    RUN chmod +x scripts/coverage-gate.sh
    RUN bash scripts/coverage-gate.sh

# lint runs go vet and other linting
lint:
    FROM +deps
    
    COPY --dir cmd pkg ./
    COPY go.mod go.sum ./
    
    RUN go vet ./...
    RUN go fmt ./...

# image builds the container image
image:
    FROM go+base-go-runtime
    
    USER nonroot
    WORKDIR /app
    
    # Copy binary from build stage
    COPY +build/crossplane-plan /app/crossplane-plan
    
    ENTRYPOINT ["/app/crossplane-plan"]
    
    # Save with tag
    ARG tag=latest
    SAVE IMAGE crossplane-plan:${tag}

# publish pushes the image to ghcr.io
publish:
    FROM +image
    
    ARG tag=latest
    
    # Push to ghcr (authenticate with: docker login ghcr.io)
    SAVE IMAGE --push ghcr.io/millstonehq/crossplane-plan:${tag}

# all runs all checks and builds
all:
    BUILD +test
    BUILD +lint
    BUILD +build
