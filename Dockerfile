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
WORKDIR /build/cmd/book
RUN go build 
WORKDIR /build/cmd/session
RUN go build  
WORKDIR /build/cmd/shell
RUN go build 

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/cmd/book/book .
RUN cp /build/cmd/session/session .
RUN cp /build/cmd/shell/shell .

# Build a small image for book
FROM scratch as book

COPY --from=builder /dist/book /book

EXPOSE 8080

# Command to run
ENTRYPOINT ["/book", "serve"]

# Build a small image for session
FROM scratch as session

COPY --from=builder /dist/session /session

EXPOSE 8082
EXPOSE 8083

# Command to run
ENTRYPOINT ["/session", "relay"]

# Build a small image for shell

FROM scratch as shell

# avoid naming conflict with linux shell
COPY --from=builder /dist/shell /shellrelay

EXPOSE 8080
EXPOSE 8081

# Command to run
ENTRYPOINT ["/shellrelay", "relay"]