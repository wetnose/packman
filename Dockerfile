FROM --platform=linux/amd64 docker.artifactory.openintegration.inc/golang:1.24.1-bullseye as builder
ARG VER=dev

WORKDIR /app

# Copy the source code
COPY file/ ./file/
COPY script/ ./script/
COPY test/ ./test/
COPY go.mod go.sum *.go ./

# Tests
RUN go test

# Build bin (MacOS)
RUN GOOS=darwin \
    go build \
    -trimpath -gcflags=-trimpath=/app -asmflags=-trimpath=/app \
    -ldflags="-s -w -X main.version=${VER}" \
    -v -o bin/macos/packman .

# Build bin (Linux)
RUN GOOS=linux \
    go build \
    -trimpath -gcflags=-trimpath=/app -asmflags=-trimpath=/app \
    -ldflags="-s -w -X main.version=${VER}" \
    -v -o bin/linux/packman .

# Build bin (Windows)
RUN GOOS=windows \
    go build \
    -trimpath -gcflags=-trimpath=/app -asmflags=-trimpath=/app \
    -ldflags="-s -w -X main.version=${VER}" \
    -v -o bin/windows/packman.exe .

# Export binaries
FROM scratch as binaries
COPY --from=builder /app/bin/ /
