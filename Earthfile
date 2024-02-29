VERSION 0.8

ARG --global GIT_TAG
ARG --global BINARY_NAME="gha-exporter"
ARG --global IMAGE_NAME="gha-exporter"

# This target is used to setup a common Go environment used for both builds and tests.
go-environment:
    # Environment variables
    ARG --required TARGETOS
    ARG --required TARGETARCH
    # Native arch is required because otherewise images will default to TARGETARCH,
    # which is overridden by `--platform`.
    ARG --required NATIVEARCH
    
    # This keeps the Go version set in a single place
    # A container is used to pin the `sed` dependency. `LOCALLY` could be used instead, but is
    # disallowed by the `--strict` Earthly flag which is used to help enfore reproducability.
    FROM --platform="linux/$NATIVEARCH" alpine:3.19.0
    WORKDIR /gomod
    COPY go.mod .
    LET GO_VERSION=$(sed -rn 's/^go (.*)$/\1/p' go.mod)
    
    # Run on the native architecture, but setup for cross compilation.
    FROM --platform="linux/$NATIVEARCH" "golang:$GO_VERSION"
    ENV GOOS=$TARGETOS
    ENV GOARCH=$TARGETARCH
    WORKDIR /go/src
    CACHE --sharing shared $(go env GOMODCACHE)

    # Load the source and download modules
    COPY . .
    RUN go mod download -x

# Produces a single executable binary file for the target platform.
# This should generally be called as `earthly --platform=<output binary platform> +binary`.
binary:
    ARG --required TARGETPLATFORM
    FROM --platform "$TARGETPLATFORM" +go-environment
    # Caches are specific to a given target, so the GOCACHE is declared here as it
    # is updated when builds run
    CACHE --sharing shared --id gocache $(go env GOCACHE)

    # Setup for the build
    LET LINKER_FLAGS="-s -w"
    IF [ -n "$GIT_TAG" ]
        ARG EARTHLY_GIT_SHORT_HASH
        SET LINKER_FLAGS="$LINKER_FLAGS -X 'main.Version=$GIT_TAG+$EARTHLY_GIT_SHORT_HASH'"
    END
    LET BINARY_OUTPUT_PATH="../$BINARY_NAME"

    # Do the actual build
    RUN go build -o "$BINARY_OUTPUT_PATH" -ldflags="$LINKER_FLAGS" cmd/main.go

    # Process the outputs
    SAVE ARTIFACT "$BINARY_OUTPUT_PATH" AS LOCAL "outputs/$TARGETPLATFORM/$BINARY_NAME"

# Same as `binary`, except the platform defaults to the local host.
local-binary:
    # This is a workaround for the default TARGETOS value being the buildkit OS (linux),
    # which is wrong when running on MacOS. Unfortunately this is not fixed by specifying
    # TARGETOS under LOCALLY, because then it cannot be overridden via the platform arg.
    ARG --required USERPLATFORM
    BUILD --platform="$USERPLATFORM" +binary

# Produces a container image and multiarch manifest. These are automatically loaded into the
# local Docker image cache. If multiple platforms are specified, then they are all added
# under the same image.
container-image:
    # Build args
    ARG --required TARGETARCH
    ARG --required NATIVEARCH
    ARG CONTAINER_REGISTRY=""

    # Setup for build
    # `IF` statements essentially run as shell `if` statements, so a build context must be declared
    # for them.
    FROM --platform="linux/$NATIVEARCH" alpine:3.19.0
    LET IMAGE_TAG="latest"
    IF [ -n "$GIT_TAG" ]
        SET IMAGE_TAG="$GIT_TAG"
    END

    # Do the actual build
    RUN echo "FULL IMAGE NAME: $CONTAINER_REGISTRY$IMAGE_NAME:$IMAGE_TAG"
    FROM --platform="linux/$TARGETARCH" scratch
    COPY --platform="linux/$TARGETARCH" +binary/* /
    # Unfortunately arg expansion is not supported here, see https://github.com/earthly/earthly/issues/1846
    ENTRYPOINT [ "/gha-exporter" ]

    # Process the outputs
    SAVE IMAGE --push "$CONTAINER_REGISTRY$IMAGE_NAME:$IMAGE_TAG"

# Same as `binary`, but wraps the output in a tarball.
tarball:
    ARG --required TARGETOS
    ARG --required TARGETARCH
    ARG --required NATIVEARCH
    ARG TARBALL_NAME="$BINARY_NAME-$TARGETOS-$TARGETARCH.tar.gz"

    FROM --platform="linux/$NATIVEARCH" alpine:3.19.0
    WORKDIR /tarball
    COPY --platform="$TARGETOS/$TARGETARCH" +binary/* .
    RUN tar -czvf "$TARBALL_NAME" *
    SAVE ARTIFACT $TARBALL_NAME AS LOCAL outputs/$TARGETOS/$TARGETARCH/$TARBALL_NAME

local-tarball:
    ARG --required USERPLATFORM
    BUILD --platform="$USERPLATFORM" +tarball

all:
    BUILD +local-binary
    BUILD +local-tarball
    BUILD +container-image

# Runs the project's Go tests.
test:
    # Probably not needed, but this supports running tests on different architectures.
    ARG --required TARGETPLATFORM
    # For options, see
    # https://github.com/gotestyourself/gotestsum?tab=readme-ov-file#output-format
    ARG OUTPUT_FORMAT="pkgname-and-test-fails"

    FROM --platform "$TARGETPLATFORM" +go-environment
    WORKDIR /go/src
    CACHE --sharing shared $(go env GOMODCACHE)
    CACHE --sharing shared --id gocache $(go env GOCACHE)
    RUN go install gotest.tools/gotestsum@latest
    RUN gotestsum --format "$OUTPUT_FORMAT" ./... -- -shuffle on -timeout 2m -race

lint:
    # For options, see https://golangci-lint.run/usage/configuration/#command-line-options
    ARG OUTPUT_FORMAT="colored-line-number"

    # Setup the linter and configure the environment
    FROM +go-environment
    WORKDIR /go/src
    ENV GOLANGCI_LINT_CACHE=/golangci-lint-cache
    CACHE $GOLANGCI_LINT_CACHE
    CACHE --sharing shared $(go env GOMODCACHE)
    CACHE --sharing shared --id gocache $(go env GOCACHE)
    RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2

    # Run the linter
    RUN golangci-lint run ./... --out-format "$OUTPUT_FORMAT"

# Removes local file and container image artifacts.
clean:
    LOCALLY

    # Delete output files
    RUN rm -rf "outputs/"

    # Delete container images
    FOR IMAGE IN $(docker image ls --filter "reference=$IMAGE_NAME" --quiet | sort | uniq)
        RUN docker image rm --force "$IMAGE"
    END

# Cuts a new GH release and pushes file assets to it. Also pushes container images.
release:
    ARG --required GIT_TAG  # This global var is redeclared here to ensure that it is set via `--required`
    ARG CONTAINER_REGISTRY="ghcr.io/gravitational/gha-exporter/"

    # Create GH release and upload artifact(s)
    FROM alpine:3.19.0
    # Unfortunately GH does not release a container image for their CLI, see https://github.com/cli/cli/issues/2027
    RUN apk add github-cli
    WORKDIR /release_artifacts
    COPY --platform=linux/amd64 --platform=linux/arm64 --platform=darwin/arm64 +tarball/* .
    COPY CHANGELOG.md /CHANGELOG.md
    # Run commands with "--push" set will only run when the "--push" arg is provided via CLI
    RUN --push gh release create --draft --verify-tag --notes-file "/CHANGELOG.md" --prerelease "$GIT_TAG" "./*"

    # Build container images and push them
    BUILD --platform=linux/amd64 --platform=linux/arm64 +container-image --CONTAINER_REGISTRY="$CONTAINER_REGISTRY"
