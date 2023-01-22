FROM golang:alpine AS builder

# Set necessary environment variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the applications
WORKDIR /build/cmd/relay
RUN go build 

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/cmd/relay/relay .

# Build a small image for relay
FROM scratch as relay

COPY --from=builder /dist/relay /relay

EXPOSE 8082
EXPOSE 8083

# Command to run
ENTRYPOINT ["/relay", "relay"]

