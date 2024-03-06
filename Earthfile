VERSION 0.8

ARG --global GIT_TAG
ARG --global BINARY_NAME="gha-exporter"
ARG --global IMAGE_NAME="gha-exporter"
ARG --global USEROS
ARG --global USERARCH
ARG --global GOOS=$USEROS
ARG --global GOARCH=$USERARCH

# TODO remove this, this is a temp workaround
download-go-github:
    ARG NATIVEARCH
    FROM --platform="linux/$NATIVEARCH" alpine/git:2.43.0
    WORKDIR /src
    RUN git clone --single-branch --depth 1 --branch v60.0.0 https://github.com/google/go-github.git .
    SAVE ARTIFACT go.mod
    SAVE ARTIFACT . AS LOCAL go-github

# TODO remove this, this is a temp workaround
build-go-github-patch:
    ARG NATIVEARCH
    # Pull the Go version from the project
    FROM --platform="linux/$NATIVEARCH" alpine:3.19.0
    WORKDIR /gomod
    COPY +download-go-github/go.mod .
    LET GO_VERSION=$(sed -rn 's/^go (.*)$/\1/p' go.mod)

    FROM --platform "linux/$NATIVEARCH" "golang:$GO_VERSION"
    WORKDIR /go/src/
    COPY +download-go-github/* .

    # Download the project's requirements
    CACHE --sharing shared --id gomodcache $(go env GOMODCACHE)
    RUN go mod download -x

    # Load and apply the patch
    CACHE --sharing shared --id gocache $(go env GOCACHE)
    COPY ./hack-todo-remove/go-github.patch go-github.patch
    RUN git apply go-github.patch && GOOS=linux GOARCH=$NATIVEARCH go generate ./...
    SAVE ARTIFACT . AS LOCAL go-github

# This target is used to setup a common Go environment used for both builds and tests.
go-environment:
    ARG NATIVEARCH
    # This keeps the Go version set in a single place
    # A container is used to pin the `sed` dependency. `LOCALLY` could be used instead, but is
    # disallowed by the `--strict` Earthly flag which is used to help enfore reproducability.
    FROM --platform="linux/$NATIVEARCH" alpine:3.19.0
    WORKDIR /gomod
    COPY go.mod .
    LET GO_VERSION=$(sed -rn 's/^go (.*)$/\1/p' go.mod)
    
    # Setup Go.
    FROM --platform="linux/$NATIVEARCH" "golang:$GO_VERSION"
    WORKDIR /go/src
    CACHE --sharing shared --id gomodcache $(go env GOMODCACHE)

    # Load the source and download modules
    COPY . .
    RUN go mod download -x

    # TODO remove this, this is a temp workaround
    COPY +build-go-github-patch/* ./go-github
    RUN go mod edit -replace github.com/google/go-github/v60=./go-github

# Produces a single executable binary file for the target platform.
binary:
    FROM +go-environment
    # Caches are specific to a given target, so the GOCACHE is declared here as it
    # is updated when builds run
    CACHE --sharing shared --id gocache $(go env GOCACHE)

    # Setup for the build
    LET LINKER_FLAGS="-s -w"
    IF [ -n "$GIT_TAG" ]
        ARG EARTHLY_GIT_SHORT_HASH
        SET LINKER_FLAGS="$LINKER_FLAGS -X 'main.Version=${GIT_TAG#v}+$EARTHLY_GIT_SHORT_HASH'"
    END
    LET BINARY_OUTPUT_PATH="../$BINARY_NAME"

    # Do the actual build
    RUN go build -o "$BINARY_OUTPUT_PATH" -ldflags="$LINKER_FLAGS" .

    # Process the outputs
    SAVE ARTIFACT "$BINARY_OUTPUT_PATH" AS LOCAL "outputs/$GOOS/$GOARCH/$BINARY_NAME"

# Produces a container image and multiarch manifest. These are automatically loaded into the
# local Docker image cache. If multiple platforms are specified, then they are all added
# under the same image.
container-image:
    # Build args
    ARG TARGETARCH
    ARG NATIVEARCH
    ARG CONTAINER_REGISTRY

    # Setup for build
    # `IF` statements essentially run as shell `if` statements, so a build context must be declared
    # for them.
    FROM --platform="linux/$NATIVEARCH" alpine:3.19.0
    LET IMAGE_TAG="latest"
    IF [ -n "$GIT_TAG" ]
        SET IMAGE_TAG="${GIT_TAG#v}"
    END

    # Do the actual build
    FROM --platform="linux/$TARGETARCH" scratch
    COPY (+binary/* --GOOS="linux" --GOARCH="$TARGETARCH") /
    # Unfortunately arg expansion is not supported here, see https://github.com/earthly/earthly/issues/1846
    ENTRYPOINT [ "/gha-exporter" ]

    # Process the outputs
    SAVE IMAGE --push "$CONTAINER_REGISTRY$IMAGE_NAME:$IMAGE_TAG"

# Same as `binary`, but wraps the output in a tarball.
tarball:
    ARG NATIVEARCH
    ARG TARBALL_NAME="$BINARY_NAME-$GOOS-$GOARCH.tar.gz"

    FROM  --platform="linux/$NATIVEARCH" alpine:3.19.0
    WORKDIR /tarball
    COPY +binary/* .
    RUN tar -czvf "$TARBALL_NAME" *
    SAVE ARTIFACT $TARBALL_NAME AS LOCAL "outputs/$GOOS/$GOARCH/$TARBALL_NAME"

all:
    BUILD +binary
    BUILD +tarball
    BUILD +container-image

# Runs the project's Go tests.
test:
    ARG NATIVEARCH
    # For options, see
    # https://github.com/gotestyourself/gotestsum?tab=readme-ov-file#output-format
    ARG OUTPUT_FORMAT="pkgname-and-test-fails"

    FROM +go-environment
    WORKDIR /go/src
    CACHE --sharing shared --id gomodcache $(go env GOMODCACHE)
    CACHE --sharing shared --id gocache $(go env GOCACHE)
    RUN GOOS="linux" GOARCH="$NATIVEARCH" go install gotest.tools/gotestsum@latest
    RUN gotestsum --format "$OUTPUT_FORMAT" ./... -- -shuffle on -timeout 2m -race

lint:
    ARG NATIVEARCH
    # For options, see https://golangci-lint.run/usage/configuration/#command-line-options
    ARG OUTPUT_FORMAT="colored-line-number"

    # Setup the linter and configure the environment
    FROM +go-environment
    WORKDIR /go/src
    ENV GOLANGCI_LINT_CACHE=/golangci-lint-cache
    CACHE $GOLANGCI_LINT_CACHE
    CACHE --sharing shared --id gomodcache $(go env GOMODCACHE)
    CACHE --sharing shared --id gocache $(go env GOCACHE)
    RUN GOOS="linux" GOARCH="$NATIVEARCH" go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2

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
    ARG EARTHLY_PUSH
    ARG NATIVEARCH

    # Create GH release and upload artifact(s)
    FROM  --platform="linux/$NATIVEARCH" alpine:3.19.0

    # Unfortunately GH does not release a container image for their CLI, see https://github.com/cli/cli/issues/2027
    RUN apk add github-cli
    WORKDIR /release_artifacts
    COPY (+tarball/* --GOOS=linux --GOARCH=amd64) (+tarball/* --GOOS=linux --GOARCH=arm64) (+tarball/* --GOOS=darwin --GOARCH=arm64) .
    COPY CHANGELOG.md /CHANGELOG.md
    # Run commands with "--push" set will only run when the "--push" arg is provided via CLI
    RUN --push --secret GH_TOKEN gh release create --draft --verify-tag --notes-file "/CHANGELOG.md" --prerelease "$GIT_TAG" ./*

    # Build container images and push them
    BUILD --platform=linux/amd64 --platform=linux/arm64 +container-image --CONTAINER_REGISTRY="$CONTAINER_REGISTRY"
