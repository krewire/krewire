# `ai`

Import: `github.com/krewire/krewire/packages/ai`

## Purpose

Native AI SDK for Go applications: multi-provider LLM integrations, streaming completions, structured schema generation, object extraction, and agentic tool invocation.

## Main API

- `ai.GenerateText(ctx, model, prompt, opts)`: Direct text completion.
- `ai.StreamText(ctx, model, prompt, opts)`: Real-time chunk streaming.
- `ai.GenerateObject[T](ctx, model, prompt, opts)`: Type-safe structured JSON object extraction.
- `ai.Tool`: Tool calling definitions with input validation and execution handlers.
- `providers`: Pluggable LLM backends (OpenAI, Anthropic, Google Gemini, Ollama, local models).

## Example

```go
package main

import (
	"context"
	"fmt"
	"github.com/krewire/krewire/packages/ai"
	"github.com/krewire/krewire/packages/ai/providers/gemini"
)

type Summary struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

func main() {
	model := gemini.New("gemini-1.5-flash")
	obj, err := ai.GenerateObject[Summary](context.Background(), model, "Analyze Krewire framework")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Title: %s, Tags: %v\n", obj.Title, obj.Tags)
}
```

## Design boundary

Independent domain package. Zero coupling to HTTP transport or CLI devtools.
