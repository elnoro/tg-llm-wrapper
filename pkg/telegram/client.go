package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const defaultTimeout = 60

var errNotOk = errors.New("response is not ok")

type Client struct {
	baseURL string
	timeout int
	debug   bool

	client *http.Client
}

func NewTelegramBotFromToken(token string, debug bool) *Client {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/", token)

	return &Client{
		baseURL: url,
		timeout: defaultTimeout,
		client:  &http.Client{},
		debug:   debug,
	}
}

func (b *Client) GetMe(ctx context.Context) (Me, error) {
	resp, err := b.SendRequest(ctx, "getMe", "")
	if err != nil {
		return Me{}, fmt.Errorf("failed to get me: %w", err)
	}

	var me Me
	err = json.Unmarshal([]byte(resp), &me)
	if err != nil {
		return Me{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !me.Ok {
		return Me{}, fmt.Errorf("response is not ok: %w", errNotOk)
	}

	return me, nil
}

func (b *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	message := message{
		ChatID: chatID,
		Text:   text,
	}

	req, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = b.SendRequest(ctx, "sendMessage", string(req))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

type message struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func (b *Client) GetUpdates(ctx context.Context, offset int64) (NewUpdates, error) {
	req := fmt.Sprintf(
		`{"timeout": %d, "offset": %d, "allowed_updates": ["message"]}`,
		b.timeout,
		offset,
	)
	resp, err := b.SendRequest(ctx, "getUpdates", req)
	if err != nil {
		return NewUpdates{}, fmt.Errorf("failed to get updates: %w", err)
	}

	var updates NewUpdates
	err = json.Unmarshal([]byte(resp), &updates)
	if err != nil {
		return NewUpdates{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !updates.Ok {
		return NewUpdates{}, fmt.Errorf("response is not ok: %w", errNotOk)
	}

	return updates, nil
}

func (b *Client) GetUpdatesChan(ctx context.Context, offset int64) (<-chan Update, error) {
	updatesChan := make(chan Update)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(updatesChan)
				return
			default:
				updates, err := b.GetUpdates(ctx, offset)
				if err != nil && !errors.Is(err, context.Canceled) {
					slog.Error("failed to get updates", slog.String("error", err.Error()))
					continue
				}

				for _, update := range updates.Result {
					if update.UpdateId >= offset {
						offset = update.UpdateId + 1
					}
					updatesChan <- update
				}
			}
		}
	}()

	return updatesChan, nil
}

func (b *Client) SendChatAction(ctx context.Context, chatID int64, action string) error {
	req := fmt.Sprintf(`{"chat_id": %d, "action": "%s"}`, chatID, action)

	_, err := b.SendRequest(ctx, "sendChatAction", req)
	if err != nil {
		return fmt.Errorf("failed to send chat action: %w", err)
	}

	return nil
}

func (b *Client) SendRequest(ctx context.Context, url, body string) (string, error) {
	if b.debug {
		slog.Info("sending request", slog.String("url", url), slog.String("body", body))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if b.debug {
		slog.Info("received response", slog.String("body", string(respBody)))
	}

	return string(respBody), nil
}

type Me struct {
	Ok     bool `json:"ok"`
	Result struct {
		Id                      int64  `json:"id"`
		IsBot                   bool   `json:"is_bot"`
		FirstName               string `json:"first_name"`
		Username                string `json:"username"`
		CanJoinGroups           bool   `json:"can_join_groups"`
		CanReadAllGroupMessages bool   `json:"can_read_all_group_messages"`
		SupportsInlineQueries   bool   `json:"supports_inline_queries"`
	} `json:"result"`
}

type NewUpdates struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type Update struct {
	UpdateId int64 `json:"update_id"`
	Message  struct {
		MessageId int64 `json:"message_id"`
		From      struct {
			Id           int64  `json:"id"`
			IsBot        bool   `json:"is_bot"`
			FirstName    string `json:"first_name"`
			Username     string `json:"username"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Chat struct {
			Id        int64  `json:"id"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date int64  `json:"date"`
		Text string `json:"text"`
	} `json:"message"`
}

func (u Update) IsCommand() bool {
	return u.Message.Text[0] == '/'
}
