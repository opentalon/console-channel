package consolechannel

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/opentalon/opentalon/pkg/channel"
)

// ID is the channel identifier used for the console (stdin/stdout).
const ID = "console"

// Channel is a built-in channel that reads user input from stdin and
// writes assistant replies to stdout. Used to run OpenTalon in the terminal.
//
// conversationID is minted once per process and stamped on every
// InboundMessage. The orchestrator combines it with the authenticated
// user_id (resolved from profile_token via whoami) to form the session_id
// `<user>:console:<conversationID>`, so each process restart starts a
// fresh session row rather than collapsing every terminal invocation into
// a single stable `<user>:console:console` row. Mirrors the websocket-
// channel pattern where each accept() mints its own convID.
type Channel struct {
	mu             sync.Mutex
	done           chan struct{}
	closed         bool
	profileToken   string
	conversationID string
}

// New returns a channel that uses stdin/stdout for I/O. Identity is
// supplied later via Configure (see ConfigurableChannel in
// opentalon/pkg/channel — websocket-channel uses the same pattern for
// its server addr/path/CORS config); New itself captures no
// environment or filesystem state.
func New() *Channel {
	return &Channel{
		done:           make(chan struct{}),
		conversationID: newID(),
	}
}

// Configure implements channel.ConfigurableChannel. The host (OpenTalon
// Core) invokes this with the `config:` block from channel YAML before
// Start. Supported keys:
//
//   - profile_token (string): bearer the orchestrator resolves via
//     profiles.who_am_i to derive entity_id + group, exactly like the
//     ?token= query param on websocket-channel. When unset the channel
//     is anonymous and downstream identity-scoped consumers (e.g.
//     tenant-scoped session listings) will not see these sessions.
//     Typically wired in YAML as
//     `profile_token: "${OPENTALON_CONSOLE_PROFILE_TOKEN}"` so the
//     value lives in a .env file rather than the committed config.
func (c *Channel) Configure(config map[string]interface{}) error {
	if v, ok := config["profile_token"].(string); ok {
		c.profileToken = v
	}
	return nil
}

// ID implements channel.Channel.
func (c *Channel) ID() string { return ID }

// Capabilities implements channel.Channel.
func (c *Channel) Capabilities() channel.Capabilities {
	return channel.Capabilities{
		ID:               ID,
		Name:             "Console",
		Threads:          false,
		Files:            false,
		Reactions:        false,
		Edits:            false,
		MaxMessageLength: 64 * 1024,
	}
}

// Start implements channel.Channel. It prints the console banner, then starts a
// goroutine that reads lines from stdin and sends them as InboundMessage to inbox.
// When context is cancelled or stdin hits EOF, the goroutine closes the inbox and returns.
func (c *Channel) Start(ctx context.Context, inbox chan<- channel.InboundMessage) error {
	banner := "OpenTalon. Type a message and press Enter. Try 'hello' to run the hello-world plugin. Ctrl+D or Ctrl+C to exit."
	if c.profileToken == "" {
		banner += "\n[console-channel] profile_token not configured — sessions will be anonymous (no identity resolution)."
	}
	_, _ = fmt.Fprint(os.Stdout, banner+"\n\n")
	go c.readLoop(ctx, inbox)
	return nil
}

func (c *Channel) readLoop(ctx context.Context, inbox chan<- channel.InboundMessage) {
	defer close(inbox)

	scanner := bufio.NewScanner(os.Stdin)
	lineCh := make(chan string, 1)

	go func() {
		for scanner.Scan() {
			lineCh <- scanner.Text()
		}
		close(lineCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-lineCh:
			if !ok {
				return
			}
			trimmed := trimLine(line)
			if trimmed == "" {
				continue
			}
			msg := channel.InboundMessage{
				ChannelID:      ID,
				ConversationID: c.conversationID,
				ThreadID:       "",
				SenderID:       "user",
				SenderName:     "user",
				Content:        trimmed,
				Timestamp:      time.Now(),
			}
			if c.profileToken != "" {
				msg.Metadata = map[string]string{"profile_token": c.profileToken}
			}
			select {
			case <-ctx.Done():
				return
			case inbox <- msg:
			}
		}
	}
}

func trimLine(s string) string {
	b := 0
	for b < len(s) && (s[b] == ' ' || s[b] == '\t') {
		b++
	}
	e := len(s)
	for e > b && (s[e-1] == ' ' || s[e-1] == '\t' || s[e-1] == '\r') {
		e--
	}
	return s[b:e]
}

// Send implements channel.Channel. It writes the message content to stderr so it's visible when running as a subprocess (stdout may be used for protocol).
func (c *Channel) Send(ctx context.Context, msg channel.OutboundMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	// Ensure response is on its own line and visible (flush so user sees LLM answer).
	_, _ = os.Stderr.WriteString("\n")
	_, _ = os.Stderr.WriteString(msg.Content)
	if len(msg.Content) == 0 || msg.Content[len(msg.Content)-1] != '\n' {
		_, _ = os.Stderr.WriteString("\n")
	}
	_ = os.Stderr.Sync()
	return nil
}

// Stop implements channel.Channel.
func (c *Channel) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	close(c.done)
	return nil
}

// newID returns a fresh 32-char hex string for the per-process
// ConversationID. Mirrors websocket-channel's newID — keeping the format
// identical means session_id parsing on the orchestrator side stays
// uniform across channels.
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
