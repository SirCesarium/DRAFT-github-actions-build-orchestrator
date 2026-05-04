FROM golang:1.26-bookworm

# Install Rust (silent)
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs 2>/dev/null | sh -s -- -y > /dev/null 2>&1
ENV PATH="/root/.cargo/bin:${PATH}"

# Pre-install all required Rust targets (silent)
RUN rustup target add \
    x86_64-unknown-linux-gnu \
    x86_64-unknown-linux-musl \
    i686-unknown-linux-gnu \
    aarch64-unknown-linux-gnu \
    wasm32-wasip1 \
    x86_64-pc-windows-gnu \
    i686-pc-windows-gnu \
    x86_64-apple-darwin \
    aarch64-apple-darwin > /dev/null 2>&1

# Install multi-arch toolchains, emulators and dependencies (silent)
RUN dpkg --add-architecture i386 > /dev/null 2>&1 && \
    apt-get update -qq > /dev/null 2>&1 && \
    apt-get install -y -qq \
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
    > /dev/null 2>&1 && \
    rm -rf /var/lib/apt/lists/*

# Install Wasmtime (silent)
RUN curl -s 'https://wasmtime.dev/install.sh' 2>/dev/null | bash > /dev/null 2>&1
ENV PATH="/root/.wasmtime/bin:${PATH}"

# Install cargo subcommands (silent)
RUN cargo install cargo-deb cargo-generate-rpm > /dev/null 2>&1 || true

WORKDIR /app

COPY . .

# Copy LICENSE locally to the smoke projects
RUN cp LICENSE tests/smoke/rust-project/LICENSE

# Warm up Go dependencies
RUN go mod download

# Pre-build Go project to cache dependencies
RUN cd tests/smoke/go-project && go build -o /dev/null .

CMD ["make", "smoke-test-local"]
