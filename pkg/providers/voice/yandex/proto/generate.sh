#!/bin/bash

# Generate Go code from proto files for Yandex TTS

set -e

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed. Please install Protocol Buffers compiler."
    echo "On macOS: brew install protobuf"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Error: protoc-gen-go is not installed."
    echo "Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Error: protoc-gen-go-grpc is not installed."
    echo "Install with: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "Generating Go code from proto files..."

# Create output directory
mkdir -p generated

# Generate for TTS
protoc \
    --go_out=generated \
    --go_opt=paths=source_relative \
    --go-grpc_out=generated \
    --go-grpc_opt=paths=source_relative \
    -I. \
    tts/tts.proto \
    tts/tts_service.proto

# Generate for STT
protoc \
    --go_out=generated \
    --go_opt=paths=source_relative \
    --go-grpc_out=generated \
    --go-grpc_opt=paths=source_relative \
    -I. \
    stt/package_options.proto \
    stt/stt.proto \
    stt/stt_service.proto

echo "Proto generation complete!"
echo "Generated files are in: $SCRIPT_DIR/generated"
