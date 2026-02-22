package consolechannel

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/opentalon/opentalon/pkg/channel"
)

func TestTrimLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"no trim", "hello", "hello"},
		{"leading space", "  hello", "hello"},
		{"trailing space", "hello  ", "hello"},
		{"leading tab", "\thello", "hello"},
		{"trailing cr", "hello\r", "hello"},
		{"both ends", "  hello \t\r", "hello"},
		{"only whitespace", "   \t  ", ""},
		{"inner spaces kept", "hello world", "hello world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimLine(tt.in)
			if got != tt.want {
				t.Errorf("trimLine(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestID(t *testing.T) {
	if id := ID; id != "console" {
		t.Errorf("ID = %q, want \"console\"", id)
	}
}

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.ID() != "console" {
		t.Errorf("ID() = %q, want \"console\"", c.ID())
	}
}

func TestCapabilities(t *testing.T) {
	c := New()
	caps := c.Capabilities()
	if caps.ID != "console" {
		t.Errorf("Capabilities().ID = %q, want \"console\"", caps.ID)
	}
	if caps.Name != "Console" {
		t.Errorf("Capabilities().Name = %q, want \"Console\"", caps.Name)
	}
	if caps.Threads != false || caps.Files != false || caps.Reactions != false || caps.Edits != false {
		t.Errorf("Capabilities() flags should be false: Threads=%v Files=%v Reactions=%v Edits=%v",
			caps.Threads, caps.Files, caps.Reactions, caps.Edits)
	}
	if caps.MaxMessageLength != 64*1024 {
		t.Errorf("Capabilities().MaxMessageLength = %d, want 65536", caps.MaxMessageLength)
	}
}

func TestStop(t *testing.T) {
	c := New()
	if err := c.Stop(); err != nil {
		t.Errorf("Stop() = %v", err)
	}
	// Second Stop is a no-op
	if err := c.Stop(); err != nil {
		t.Errorf("Stop() second call = %v", err)
	}
}

func TestSend(t *testing.T) {
	// Redirect stderr so we don't mix with test output
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = old }()

	c := New()
	ctx := context.Background()
	msg := channel.OutboundMessage{Content: "test reply"}

	if err := c.Send(ctx, msg); err != nil {
		t.Errorf("Send() = %v", err)
	}

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()
	if out != "\ntest reply\n" {
		t.Errorf("Send wrote %q, want \"\\ntest reply\\n\"", out)
	}
}

func TestSend_afterStop(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = old }()

	c := New()
	_ = c.Stop()

	if err := c.Send(context.Background(), channel.OutboundMessage{Content: "no"}); err != nil {
		t.Errorf("Send after Stop() = %v, want nil", err)
	}

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if buf.Len() != 0 {
		t.Errorf("Send after Stop should write nothing, wrote %q", buf.String())
	}
}

func TestSend_emptyContent_addsNewline(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = old }()

	c := New()
	_ = c.Send(context.Background(), channel.OutboundMessage{Content: ""})
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if buf.String() != "\n\n" {
		t.Errorf("Send empty content wrote %q, want \"\\n\\n\"", buf.String())
	}
}
