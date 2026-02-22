package consolechannel

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/opentalon/opentalon/pkg/channel"
)

func init() {
	channel.RegisterContentPreparer(ID, PrepareContent)
}

// PrepareContent runs channel-specific pre-processing on user content before
// it is sent to the orchestrator. For the console, when the user types "hello",
// we call the hello-world plugin's prepare action (concat " world" + random
// prompt fragment) and use the result as the content for the LLM.
func PrepareContent(
	ctx context.Context,
	content string,
	runAction channel.RunActionFunc,
	hasAction channel.HasActionFunc,
) string {
	if !strings.Contains(strings.ToLower(content), "hello") || !hasAction("hello-world", "prepare") {
		return content
	}
	out, err := runAction(ctx, "hello-world", "prepare", map[string]string{"text": content})
	if err != nil {
		return content
	}
	var parsed struct {
		Text           string `json:"text"`
		PromptFragment string `json:"prompt_fragment"`
	}
	if json.Unmarshal([]byte(out), &parsed) != nil || parsed.Text == "" {
		return content
	}
	content = parsed.Text
	if parsed.PromptFragment != "" {
		content = parsed.PromptFragment + "\n\nUser message: " + content
	}
	return content
}
