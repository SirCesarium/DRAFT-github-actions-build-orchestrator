FROM golang:1.26-bookworm

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Pre-install all required Rust targets
RUN rustup target add \
    x86_64-unknown-linux-gnu \
    x86_64-unknown-linux-musl \
    i686-unknown-linux-gnu \
    aarch64-unknown-linux-gnu \
    wasm32-wasip1 \
    x86_64-pc-windows-gnu \
    i686-pc-windows-gnu \
    x86_64-apple-darwin \
    aarch64-apple-darwin

# Install multi-arch toolchains, emulators and dependencies
RUN dpkg --add-architecture i386 && apt-get update && apt-get install -y \
    build-essential \
    gcc-aarch64-linux-gnu \
    gcc-i686-linux-gnu \
    gcc-mingw-w64 \
    musl-tools \
    libc6-dev-i386 \
    qemu-user \
    qemu-system-arm \
    wine \
    wine64 \
    msitools \
    file \
    rpm \
    cpio \
    bc \
    && rm -rf /var/lib/apt/lists/*

# Install Wasmtime
RUN curl 'https://wasmtime.dev/install.sh' -sSf | bash
ENV PATH="/root/.wasmtime/bin:${PATH}"

# Install cargo subcommands
RUN cargo install cargo-deb cargo-generate-rpm || true

WORKDIR /app

COPY . .

# Copy LICENSE locally to the smoke projects
RUN cp LICENSE tests/smoke/rust-project/LICENSE

# Warm up Go dependencies
RUN go mod download

# Pre-build Go project to cache dependencies
RUN cd tests/smoke/go-project && go build -o /dev/null .

CMD ["make", "smoke-test-local"]
