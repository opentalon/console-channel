# Console Channel

[![CI](https://github.com/opentalon/console-channel/actions/workflows/ci.yml/badge.svg)](https://github.com/opentalon/console-channel/actions/workflows/ci.yml)

Standalone Go module: channel that runs OpenTalon in the terminal — **stdin** for user input, **stderr** for assistant replies. Used for local, interactive use.

Can be used as a **standalone repo** (own `go.mod`) or as a subdirectory of the main OpenTalon repo.

## Standalone use

Clone and build:

```bash
git clone https://github.com/opentalon/console-channel.git
cd console-channel
make build   # → binary named "console", tell opentalon config path to this binary
```

Run tests and lint:

```bash
make test    # go test -race ./...
make lint    # golangci-lint run
```

The binary is meant to be **started by the OpenTalon core** as a subprocess; it expects `OPENTALON_CHANNEL_SOCK_DIR` to be set. For interactive use, run the full OpenTalon app and enable this channel (see below).

## How it works

1. **OpenTalon core** starts the console channel as a **subprocess** (the binary you build from this module).
2. The core passes `OPENTALON_CHANNEL_SOCK_DIR` so the channel creates a **Unix socket** in that directory. No handshake is printed to stdout, so the process keeps **stdin/stdout** for the terminal.
3. The channel **reads lines from stdin** and sends each line as an `InboundMessage` over the socket to the core.
4. The core runs the orchestrator (LLM + tools) and sends the response back over the socket.
5. The channel **writes the response to stderr** (with a leading newline and flush) so the user sees the reply.

So: you type in the terminal → channel sends to core → core replies → channel prints to stderr.

## How to add it to OpenTalon

In your OpenTalon `config.yaml`, reference the channel from GitHub:

```yaml
channels:
  console:
    enabled: true
    github: "opentalon/console-channel"
    ref: "master"
    config:
      # Optional. When set, every InboundMessage carries
      # Metadata["profile_token"] = <this value>, and the orchestrator
      # resolves identity via profiles.who_am_i (entity_id + group). When
      # unset the channel is anonymous and downstream identity-scoped
      # consumers (e.g. tenant-scoped session listings) will not see
      # these sessions. Most deployments leave the literal value in a
      # `.env` file:
      profile_token: "${OPENTALON_CONSOLE_PROFILE_TOKEN}"
```

The first run clones the repo, builds the binary, and pins the resolved commit in `channels.lock`. Requires `git` and `go` on the host.

Then run OpenTalon:

```bash
./opentalon -config config.yaml
```

You get an interactive prompt; type a message, press Enter, get the LLM response. Ctrl+C or Ctrl+D to exit.

## Spec

| Item | Value |
|------|--------|
| **ID** | `console` |
| **Name** | Console |
| **Threads** | false |
| **Files** | false |
| **Reactions** | false |
| **Edits** | false |
| **MaxMessageLength** | 64 KiB |

**Protocol**

- When run by the core: `OPENTALON_CHANNEL_SOCK_DIR` is set; the channel creates `channel.sock` in that dir and waits for one connection. Methods: `capabilities`, `start`, `send`.
- **Start**: prints banner to stdout, then reads stdin line-by-line and sends each non-empty line as an inbound message to the core.
- **Send**: receives an outbound message from the core and writes `msg.Content` to **stderr** (with newline and flush).

**Entrypoint**

- Binary: `cmd/console`. It builds a `Channel` from this package and runs `channel.Serve(ctx, ch)`.

## Build and test

```bash
make build   # → binary named "console" (or set BINARY_NAME)
make test    # go test -race ./...
make lint    # golangci-lint run
```
