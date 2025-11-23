# Yandex SpeechKit Proto Definitions

This directory contains the Protocol Buffer definitions for Yandex SpeechKit TTS API v3.

## Structure

- `tts/` - TTS (Text-to-Speech) proto definitions
  - `tts.proto` - Main TTS message definitions
  - `tts_service.proto` - TTS service definitions
- `generated/` - Generated Go code from proto files

## Generating Go Code

To regenerate the Go code from proto files:

```bash
./generate.sh
```

### Prerequisites

- `protoc` - Protocol Buffers compiler
- `protoc-gen-go` - Go plugin for protoc
- `protoc-gen-go-grpc` - gRPC plugin for protoc

Install the Go plugins:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Usage

Import the generated types in your Go code:

```go
import tts "chat-ws-service/internal/infrastructure/providers/voice/yandex/proto/tts"
```

## Changes from Official Yandex Protos

1. Updated `go_package` option to use local package path
2. Removed `google/api/annotations.proto` dependency (not needed for gRPC client)
3. Removed HTTP annotations from service definitions
