#!/bin/bash
set -e

REFINERY_BIN=$(realpath "$1")
FILTER_OS="$2"
[ -f "$REFINERY_BIN" ] || (echo "Refinery bin not found" && exit 1)

# Test Rust project
cd tests/smoke/rust-project
rm -rf dist logs
mkdir -p logs

# Use strings to store PIDs and metadata for cross-shell compatibility
PIDS=""
NAMES=""
LOGS=""

add_pid() { PIDS="$PIDS $1"; }
add_name() { NAMES="$NAMES $1"; }
add_log() { LOGS="$LOGS $1"; }

# Detect current OS
CURRENT_OS=""
UNAME_S=$(uname -s)
if [[ "$UNAME_S" == "Linux" ]]; then
    CURRENT_OS="linux"
elif [[ "$UNAME_S" == "Darwin" ]]; then
    CURRENT_OS="darwin"
elif [[ "$UNAME_S" == "MINGW"* || "$UNAME_S" == "CYGWIN"* || "$UNAME_S" == "MSYS"* ]]; then
    CURRENT_OS="windows"
fi

# Run tests for current OS only (matrix strategy)
if [ -z "$FILTER_OS" ]; then
    FILTER_OS="$CURRENT_OS"
fi

for os in linux windows darwin wasi; do
    if [[ "$FILTER_OS" != *"$os"* ]]; then
        continue
    fi

    targets=""
    if [ "$os" = "linux" ]; then
        targets="x86_64:gnu x86_64:musl i686:gnu aarch64:gnu"
    elif [ "$os" = "windows" ]; then
        targets="x86_64:msvc i686:msvc x86_64:gnu i686:gnu"
    elif [ "$os" = "darwin" ]; then
        targets="x86_64 aarch64"
    elif [ "$os" = "wasi" ]; then
        targets="wasm32"
    else
        continue
    fi

    for arch_abi in $targets; do
        arch=$(echo "$arch_abi" | cut -d: -f1)
        abi=$(echo "$arch_abi" | cut -d: -f2 -s)
        
        EXTRA_RUSTFLAGS=""
        if [ "$os" = "windows" ]; then
            EXTRA_RUSTFLAGS="-C link-args=-static"
        fi
        
        # Build binary artifact
        name_bin="bin-$os-$arch-$abi"
        args="--artifact smoke-bin --os $os --arch $arch"
        if [ -n "$abi" ]; then
            args="$args --abi $abi"
        fi
        RUSTFLAGS="$EXTRA_RUSTFLAGS" "$REFINERY_BIN" build $args > "logs/$name_bin.log" 2>&1 &
        add_pid $!
        add_log "logs/$name_bin.log"
        add_name "$name_bin"
        
        # Build library artifact
        name_lib="lib-$os-$arch-$abi"
        args_lib="--artifact smoke-lib --os $os --arch $arch"
        if [ -n "$abi" ]; then
            args_lib="$args_lib --abi $abi"
        fi
        RUSTFLAGS="$EXTRA_RUSTFLAGS" "$REFINERY_BIN" build $args_lib > "logs/$name_lib.log" 2>&1 &
        add_pid $!
        add_log "logs/$name_lib.log"
        add_name "$name_lib"
    done
done

echo "Waiting for builds to complete..."
FAILED=0
i=0
PID_LIST=($PIDS)
NAME_LIST=($NAMES)
LOG_LIST=($LOGS)

for pid in "${PID_LIST[@]}"; do
    if ! wait "$pid"; then
        echo "FAILED: ${NAME_LIST[$i]}"
        cat "${LOG_LIST[$i]}"
        FAILED=1
    fi
    i=$((i + 1))
done

if [ $FAILED -eq 1 ]; then
    exit 1
fi

# Verify generated header
HEADER="dist/smoke-test.h"
grep -q "smoke_test_fn" "$HEADER" && echo "Header content: VALID"

check_exec() {
    local bin=$1
    local cmd=$2
    local full_path=$(realpath "dist/$bin")
    echo -n "Testing execution of $bin... "
    if [ -f "$full_path" ]; then
        local output
        output=$(LC_ALL=C $cmd "$full_path" 2>&1) || {
            echo "FAILED (Runtime error: exit code $?)"
            echo "$output"
            exit 1
        }
        echo "$output" | grep -q "Refinery Smoke Test: Success" && echo "PASSED" || (echo "FAILED (Output mismatch)" && exit 1)
    else
        echo "FAILED (Missing binary)" && exit 1
    fi
}

# Execution tests
if [ -z "$FILTER_OS" ] || [[ "$FILTER_OS" == *"linux"* ]]; then
    check_exec "smoke-bin-linux-x86_64-gnu" ""
    if [ -x "$(command -v qemu-i386)" ]; then
        check_exec "smoke-bin-linux-i686-gnu" "qemu-i386"
    fi
    if [ -x "$(command -v qemu-aarch64)" ]; then
        check_exec "smoke-bin-linux-aarch64-gnu" "qemu-aarch64 -L /usr/aarch64-linux-gnu"
    fi
fi

if [ -z "$FILTER_OS" ] || [[ "$FILTER_OS" == *"wasi"* ]]; then
    if [ -x "$(command -v wasmtime)" ]; then
        check_exec "smoke-bin-wasi-wasm32.wasm" "wasmtime"
    fi
fi

if [ -z "$FILTER_OS" ] || [[ "$FILTER_OS" == *"windows"* ]]; then
    [ -f "dist/smoke-bin-windows-x86_64-msvc.exe" ] && echo "smoke-bin-windows-x86_64-msvc.exe: OK"
    [ -f "dist/smoke-bin-windows-i686-msvc.exe" ] && echo "smoke-bin-windows-i686-msvc.exe: OK"
fi

# Package existence check
if [ -z "$FILTER_OS" ] || [[ "$FILTER_OS" == *"linux"* ]]; then
    [ -f "dist/smoke-bin-0.1.0-linux-x86_64-gnu.deb" ] && echo ".deb: OK"
    [ -f "dist/smoke-bin-0.1.0-linux-x86_64-gnu.rpm" ] && echo ".rpm: OK"
fi

echo "All Rust smoke tests PASSED!"

# Test Go project
echo "Testing Go project..."
cd "$(git rev-parse --show-toplevel)"
cd tests/smoke/go-project
rm -rf dist

# Build for linux/amd64
"$REFINERY_BIN" build --artifact app --os linux --arch amd64
[ -f "dist/app-linux-amd64" ] && echo "Go linux/amd64 build: OK" || (echo "Go linux/amd64 build: FAILED" && exit 1)

# Build for windows/amd64
"$REFINERY_BIN" build --artifact app --os windows --arch amd64
[ -f "dist/app-windows-amd64.exe" ] && echo "Go windows/amd64 build: OK" || (echo "Go windows/amd64 build: FAILED" && exit 1)

# Build for darwin/amd64
"$REFINERY_BIN" build --artifact app --os darwin --arch amd64
[ -f "dist/app-darwin-amd64" ] && echo "Go darwin/amd64 build: OK" || (echo "Go darwin/amd64 build: FAILED" && exit 1)

# Test packaging
"$REFINERY_BIN" build --artifact app --os linux --arch amd64
[ -f "dist/app-0.0.0-linux-amd64.tar.gz" ] && echo "Go linux/amd64 package: OK" || (echo "Go linux/amd64 package: FAILED" && exit 1)

echo "All Go smoke tests PASSED!"
echo "All smoke tests PASSED!"
