package consolechannel

import (
	"bufio"
	"context"
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
type Channel struct {
	mu     sync.Mutex
	done   chan struct{}
	closed bool
}

// New returns a channel that uses stdin/stdout for I/O.
func New() *Channel {
	return &Channel{done: make(chan struct{})}
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
	_, _ = fmt.Fprint(os.Stdout, "OpenTalon. Type a message and press Enter. Try 'hello' to run the hello-world plugin. Ctrl+D or Ctrl+C to exit.\n\n")
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
				ConversationID: ID,
				ThreadID:       "",
				SenderID:       "user",
				SenderName:     "user",
				Content:        trimmed,
				Timestamp:      time.Now(),
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
