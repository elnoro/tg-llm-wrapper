package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type ChatModel struct {
	llm         llms.ChatLLM
	chatHistory []schema.ChatMessage
}

func NewChatModel(llm llms.ChatLLM, systemPrompt string) (*ChatModel, error) {
	return &ChatModel{
		llm: llm,
		chatHistory: []schema.ChatMessage{
			schema.SystemChatMessage{Content: systemPrompt},
		},
	}, nil
}

func (c *ChatModel) OnJoin(name string) {
	newSystemMessage := c.chatHistory[0].GetContent() + fmt.Sprintf("\nRemember, human's name is %s", name)
	c.chatHistory[0] = schema.SystemChatMessage{Content: newSystemMessage}
}

func (c *ChatModel) Respond(ctx context.Context, userMessage string) (string, error) {
	c.chatHistory = append(c.chatHistory, schema.HumanChatMessage{Content: userMessage})
	completion, err := c.llm.Call(ctx, c.chatHistory)
	if err != nil {
		return "Sorry, assistant is unavailable right now. Try later", fmt.Errorf("failed to call LLM: %w", err)
	}

	c.chatHistory = append(c.chatHistory, completion)

	return completion.GetContent(), nil
}

func (c *ChatModel) SystemPrompt() string {
	return c.chatHistory[0].GetContent()
}

func (c *ChatModel) ChangeSystemPrompt(prompt string) {
	c.chatHistory[0] = schema.SystemChatMessage{Content: prompt}
}

func (c *ChatModel) Reset() {
	c.chatHistory = c.chatHistory[:1]
}
