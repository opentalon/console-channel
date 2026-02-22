package consolechannel

import (
	"context"
	"testing"
)

func TestPrepareContent_noHello_returnsOriginal(t *testing.T) {
	ctx := context.Background()
	runAction := func(_ context.Context, _, _ string, _ map[string]string) (string, error) {
		t.Fatal("runAction should not be called")
		return "", nil
	}
	hasAction := func(_, _ string) bool { return true }

	got := PrepareContent(ctx, "something else", runAction, hasAction)
	if got != "something else" {
		t.Errorf("PrepareContent(%q) = %q, want %q", "something else", got, "something else")
	}
}

func TestPrepareContent_hello_hasActionFalse_returnsOriginal(t *testing.T) {
	ctx := context.Background()
	runAction := func(_ context.Context, _, _ string, _ map[string]string) (string, error) {
		t.Fatal("runAction should not be called when hasAction is false")
		return "", nil
	}
	hasAction := func(_, _ string) bool { return false }

	got := PrepareContent(ctx, "hello", runAction, hasAction)
	if got != "hello" {
		t.Errorf("PrepareContent(%q) = %q, want %q", "hello", got, "hello")
	}
}

func TestPrepareContent_hello_runActionFails_returnsOriginal(t *testing.T) {
	ctx := context.Background()
	hasAction := func(plugin, action string) bool {
		return plugin == "hello-world" && action == "prepare"
	}
	runActionErr := func(_ context.Context, _, _ string, _ map[string]string) (string, error) {
		return "ignored", errFake
	}
	got := PrepareContent(ctx, "hello", runActionErr, hasAction)
	if got != "hello" {
		t.Errorf("PrepareContent when runAction errors = %q, want %q", got, "hello")
	}
}

var errFake = &errType{}

type errType struct{}

func (e *errType) Error() string { return "fake" }

func TestPrepareContent_hello_runActionInvalidJSON_returnsOriginal(t *testing.T) {
	ctx := context.Background()
	runAction := func(_ context.Context, _, _ string, _ map[string]string) (string, error) {
		return "not json", nil
	}
	hasAction := func(_, _ string) bool { return true }

	got := PrepareContent(ctx, "hello", runAction, hasAction)
	if got != "hello" {
		t.Errorf("PrepareContent with invalid JSON = %q, want %q", got, "hello")
	}
}

func TestPrepareContent_hello_runActionEmptyText_returnsOriginal(t *testing.T) {
	ctx := context.Background()
	runAction := func(_ context.Context, _, _ string, _ map[string]string) (string, error) {
		return `{"text":"","prompt_fragment":""}`, nil
	}
	hasAction := func(_, _ string) bool { return true }

	got := PrepareContent(ctx, "hello", runAction, hasAction)
	if got != "hello" {
		t.Errorf("PrepareContent with empty text = %q, want %q", got, "hello")
	}
}

func TestPrepareContent_hello_runActionValid_returnsTransformed(t *testing.T) {
	ctx := context.Background()
	runAction := func(_ context.Context, _, _ string, _ map[string]string) (string, error) {
		return `{"text":"hello world","prompt_fragment":""}`, nil
	}
	hasAction := func(plugin, action string) bool {
		return plugin == "hello-world" && action == "prepare"
	}

	got := PrepareContent(ctx, "hello", runAction, hasAction)
	if got != "hello world" {
		t.Errorf("PrepareContent = %q, want %q", got, "hello world")
	}
}

func TestPrepareContent_hello_withPromptFragment(t *testing.T) {
	ctx := context.Background()
	runAction := func(_ context.Context, _, _ string, _ map[string]string) (string, error) {
		return `{"text":"hello world","prompt_fragment":"Use a friendly tone."}`, nil
	}
	hasAction := func(_, _ string) bool { return true }

	got := PrepareContent(ctx, "hello", runAction, hasAction)
	want := "Use a friendly tone.\n\nUser message: hello world"
	if got != want {
		t.Errorf("PrepareContent = %q, want %q", got, want)
	}
}

func TestPrepareContent_HELLO_caseInsensitive(t *testing.T) {
	ctx := context.Background()
	var called bool
	runAction := func(_ context.Context, plugin, action string, _ map[string]string) (string, error) {
		called = true
		return `{"text":"HELLO world","prompt_fragment":""}`, nil
	}
	hasAction := func(_, _ string) bool { return true }

	got := PrepareContent(ctx, "HELLO", runAction, hasAction)
	if !called {
		t.Fatal("runAction was not called for HELLO")
	}
	if got != "HELLO world" {
		t.Errorf("PrepareContent = %q, want %q", got, "HELLO world")
	}
}
