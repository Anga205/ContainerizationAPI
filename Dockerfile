# syntax=docker/dockerfile:1

# Build stage: Ubuntu + Go toolchain
FROM ubuntu:24.04 AS builder

ARG DEBIAN_FRONTEND=noninteractive
ARG GO_VERSION=1.25.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl tar \
    && rm -rf /var/lib/apt/lists/*

RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then GOARCH=amd64; \
    elif [ "$ARCH" = "arm64" ]; then GOARCH=arm64; \
    else echo "Unsupported arch: $ARCH" && exit 1; fi && \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -o /tmp/go.tgz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf /tmp/go.tgz && \
    rm -f /tmp/go.tgz

ENV PATH="/usr/local/go/bin:${PATH}"
WORKDIR /src

COPY go.mod go.sum* ./
RUN if [ -f go.sum ]; then go mod download; else go mod tidy; fi

COPY . .
RUN go build -o /out/containerization-api .


# Runtime stage: Ubuntu + language runtimes/compilers needed by the API
FROM ubuntu:24.04

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        gcc \
        g++ \
        python3 \
        default-jdk-headless \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/containerization-api /app/containerization-api

EXPOSE 8080
CMD ["/app/containerization-api"]
