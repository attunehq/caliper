# x86_64 Ubuntu on an ARM Mac
FROM ubuntu:24.04

# Install necessary packages to build influxdb
# Based on: https://github.com/influxdata/influxdb/blob/main/CONTRIBUTING.md
# Build deps: build-essential pkg-config libssl-dev clang lld git protobuf-compiler python3 python3-dev python3-pip
# Runtime deps: python3 python3-pip python3-venv
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        wget \
        curl \
        build-essential \
        pkg-config \
        libssl-dev \
        clang \
        lld \
        git \
        protobuf-compiler \
        libprotobuf-dev \
        python3 \
        python3-dev \
        python3-pip \
        python3-venv && \
    rm -rf /var/lib/apt/lists/*

# Install Go
ARG GO_VERSION=1.25.4
ARG GO_ARCH=amd64

RUN wget https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-${GO_ARCH}.tar.gz && \
    rm go${GO_VERSION}.linux-${GO_ARCH}.tar.gz

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | \
    sh -s -- -y --default-toolchain stable

# Set PATH for Go and Cargo
ENV PATH="/usr/local/go/bin:/root/.cargo/bin:${PATH}"

# Set workspace
WORKDIR /workspace

CMD ["bash"]
