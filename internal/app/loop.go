package app

import (
	"context"
	"fmt"
	"github.com/elnoro/tg-llm-wrapper/internal/llm"
	"github.com/elnoro/tg-llm-wrapper/pkg/telegram"
	"log/slog"
	"strings"
	"time"
)

const (
	maxMsgLen    = 1024
	typingAction = 4 * time.Second
	startCmd     = "/start"
	systemCmd    = "/system"
	resetCmd     = "/reset"
)

type Loop struct {
	model     *llm.ChatModel
	botClient *telegram.Client
	adminId   int64

	conversationStarted bool
}

func NewLoop(model *llm.ChatModel, botClient *telegram.Client, adminId int64) *Loop {
	return &Loop{model: model, botClient: botClient, adminId: adminId}
}

func (l *Loop) Run(ctx context.Context) error {
	updates, err := l.botClient.GetUpdatesChan(ctx, 0)
	if err != nil {
		return fmt.Errorf("failed to get updates channel: %w", err)
	}

	for update := range updates {
		// independent context to provide graceful shutdown
		l.handleUpdate(context.Background(), update)
	}

	return nil
}

func (l *Loop) handleUpdate(ctx context.Context, update telegram.Update) {
	if !l.conversationStarted {
		l.model.OnJoin(update.Message.From.FirstName)
		l.conversationStarted = true
	}

	ll := slog.With(
		slog.String("message", update.Message.Text),
		slog.String("username", update.Message.From.Username),
		slog.Int64("user_id", update.Message.From.Id),
	)

	err := l.validateUpdate(update)
	if err != nil {
		ll.Error("invalid update received",
			slog.String("error", err.Error()),
			slog.Int64("admin_id", l.adminId),
		)

		return
	}

	if update.IsCommand() {
		err := l.handleCommand(ctx, update)
		if err != nil {
			ll.Error("failed to handle command", slog.String("error", err.Error()))
		}

		return
	}

	err = l.handleConversation(ctx, update)
	if err != nil {
		ll.Error("failed to handle conversation", slog.String("error", err.Error()))
	}
}

func (l *Loop) handleCommand(ctx context.Context, update telegram.Update) error {
	command := strings.Split(update.Message.Text, " ")[0]
	switch command {
	case startCmd:
		return nil
	case resetCmd:
		l.model.Reset()
		err := l.botClient.SendMessage(ctx, update.Message.Chat.Id, "Chat history reset")
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	case systemCmd:
		err := l.handleSystemCommand(ctx, update)
		if err != nil {
			return fmt.Errorf("failed to handle system command: %w", err)
		}
	default:
		err := l.botClient.SendMessage(ctx, update.Message.Chat.Id, "Unknown command")
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

func (l *Loop) handleSystemCommand(ctx context.Context, update telegram.Update) error {
	if update.Message.Text == systemCmd {
		err := l.botClient.SendMessage(ctx, update.Message.Chat.Id, l.model.SystemPrompt())
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		return nil
	}

	systemPrompt := strings.TrimSpace(update.Message.Text[len(systemCmd):])

	l.model.ChangeSystemPrompt(systemPrompt)
	slog.Info("system prompt changed")

	err := l.botClient.SendMessage(ctx, update.Message.Chat.Id, "System prompt changed")
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (l *Loop) handleConversation(ctx context.Context, update telegram.Update) error {
	err := l.botClient.SendChatAction(ctx, update.Message.Chat.Id, "typing")
	if err != nil {
		return fmt.Errorf("failed to send chat action: %w", err)
	}

	respCh := make(chan string)
	errCh := make(chan error)

	go func() {
		defer close(respCh)
		defer close(errCh)

		resp, err := l.model.Respond(ctx, update.Message.Text)
		if err != nil {
			errCh <- fmt.Errorf("failed to respond to message: %w", err)
			return
		}

		respCh <- resp
	}()

	for {
		select {
		case <-time.After(typingAction):
			err := l.botClient.SendChatAction(ctx, update.Message.Chat.Id, "typing")
			if err != nil {
				return fmt.Errorf("failed to send chat action: %w", err)
			}
		case response := <-respCh:
			err := l.botClient.SendMessage(ctx, update.Message.Chat.Id, response)
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func (l *Loop) validateUpdate(update telegram.Update) error {
	if update.Message.MessageId == 0 {
		return fmt.Errorf("telegram bot received non-message update: %s", update.Message.Text)
	}

	if update.Message.From.Id != l.adminId {
		return fmt.Errorf("telegram bot received unauthorized message: %s", update.Message.Text)
	}

	if len(update.Message.Text) > maxMsgLen {
		return fmt.Errorf("telegram bot received message that is too long: %s", update.Message.Text[:maxMsgLen])
	}
	return nil
}
