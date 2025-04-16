# Generic docker file for all tools that needs to be build
# Use an official Golang image as the base
FROM golang:1.23.8 AS builder

WORKDIR /tool

# Copy go.mod and go.sum first to leverage caching
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the entire project (including shared libraries)
COPY . .

# Build only the specific tool
RUN CGO_ENABLED=0 go build -o e2e-tests.bin ./e2e-tests

# Create a minimal runtime image
FROM alpine:latest

ARG GIT_HASH=unknown
ARG GIT_BRANCH=unknown
ARG GIT_DIRTY=""
ARG VERSION=unknown
ARG AUTHOR=unknown
LABEL git.hash=$GIT_HASH
LABEL git.branch=$GIT_BRANCH
LABEL git.dirty="$GIT_DIRTY"
LABEL version=$VERSION
LABEL author=$AUTHOR

# Create a user and group (non-root)
RUN addgroup -S geoadmin && adduser -S geoadmin -G geoadmin

# Install certificates (needed for HTTPS requests)
# RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /tool/e2e-tests.bin e2e-tests

# # Set execution permissions
RUN chmod +x e2e-tests

USER geoadmin

# Default command when running the container
ENTRYPOINT ["./e2e-tests"]
CMD []
