package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

type ChatModel struct {
	llm         llms.Model
	chatHistory []llms.MessageContent
}

func NewChatModel(llm llms.Model, systemPrompt string) (*ChatModel, error) {
	return &ChatModel{
		llm: llm,
		chatHistory: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		},
	}, nil
}

func (c *ChatModel) OnJoin(name string) {
	join := llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf("\nRemember, human's name is %s", name))
	c.chatHistory = append(c.chatHistory, join)
}

func (c *ChatModel) Respond(ctx context.Context, userMessage string) (string, error) {
	humanMessage := llms.TextParts(llms.ChatMessageTypeHuman, userMessage)
	c.chatHistory = append(c.chatHistory, humanMessage)
	completion, err := c.llm.GenerateContent(ctx, c.chatHistory)
	if err != nil {
		return "Sorry, assistant is unavailable right now. Try later", fmt.Errorf("failed to call LLM: %w", err)
	}

	resp := completion.Choices[0].Content
	aiMessage := llms.TextParts(llms.ChatMessageTypeAI, resp)
	c.chatHistory = append(c.chatHistory, aiMessage)

	return resp, nil
}

func (c *ChatModel) SystemPrompt() string {
	part := c.chatHistory[0].Parts[0]
	p, ok := part.(llms.TextContent)
	if !ok {
		return ""
	}

	return p.String()
}

func (c *ChatModel) ChangeSystemPrompt(prompt string) {
	c.chatHistory[0] = llms.TextParts(llms.ChatMessageTypeSystem, prompt)
}

func (c *ChatModel) Reset() {
	c.chatHistory = c.chatHistory[:2]
}
