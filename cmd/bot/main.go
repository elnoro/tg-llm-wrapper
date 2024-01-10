package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/elnoro/tg-llm-wrapper/internal/app"
	"github.com/elnoro/tg-llm-wrapper/internal/llm"
	custom_http "github.com/elnoro/tg-llm-wrapper/pkg/http"
	"github.com/elnoro/tg-llm-wrapper/pkg/telegram"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

type Config struct {
	Telegram struct {
		BotToken string
		AdminId  int64
		Debug    bool
	}

	LLMEngine    string // ollama or openai
	SystemPrompt string

	OLLama struct {
		Model string
		Url   string
	}
	OpenAI struct {
		ApiKey string
		Model  string
		Debug  bool
	}
}

func main() {
	cfg := Config{}

	flag.StringVar(&cfg.Telegram.BotToken, "telegram-bot-token", os.Getenv("TELEGRAM_TOKEN"), "Telegram bot token")
	var adminId int64
	adminIdFromEnv := os.Getenv("TELEGRAM_USER_ID")
	if adminIdFromEnv != "" {
		parsed, err := strconv.ParseInt(adminIdFromEnv, 10, 0)
		if err != nil {
			slog.Error("failed to convert admin id to int64", slog.String("error", err.Error()))

			return
		}

		adminId = parsed
	}

	flag.Int64Var(&cfg.Telegram.AdminId, "telegram-user-id", adminId, "Telegram user id")
	flag.BoolVar(&cfg.Telegram.Debug, "telegram-debug", false, "Debug mode for Telegram")

	flag.StringVar(&cfg.LLMEngine, "llm-engine", "openai", "LLM engine to use")
	flag.StringVar(&cfg.SystemPrompt, "system-prompt", os.Getenv("SYSTEM_PROMPT"), "custom initial prompt")

	flag.StringVar(&cfg.OLLama.Model, "ollama-model", "openhermes", "OLLama model to use. Choose here https://ollama.ai/library")
	flag.StringVar(&cfg.OLLama.Url, "ollama-url", "http://localhost:11434", "OLLama url")

	flag.StringVar(&cfg.OpenAI.ApiKey, "openai-api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key")
	flag.StringVar(&cfg.OpenAI.Model, "openai-model", "gpt-4-1106-preview", "OpenAI model to use")
	flag.BoolVar(&cfg.OpenAI.Debug, "openai-debug", false, "Debug mode for OpenAI")

	flag.Parse()

	loop := initLoop(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := loop.Run(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.Error("loop error", slog.String("error", err.Error()))
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	s := <-quit

	slog.Info("received signal, shutting down", slog.String("signal", s.String()))
	cancel()
	wg.Wait()
}

func initLoop(cfg Config) *app.Loop {
	langChainChat, err := initLangChain(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = "You are an AI assistant with a flair for friendliness and just a sprinkle of sass. " +
			"You are helpful and kind.\n" +
			"Remember, your responses are crafted to be concise," +
			" maintaining a balance between warmth, professionalism, and efficiency.\n"
	}

	chatModel, err := llm.NewChatModel(langChainChat, cfg.SystemPrompt)
	if err != nil {
		log.Fatal(err)
	}

	bot := telegram.NewTelegramBotFromToken(cfg.Telegram.BotToken, cfg.Telegram.Debug)
	me, err := bot.GetMe(context.Background())

	if err != nil {
		slog.Error("telegram bot authorization failed", slog.String("error", err.Error()))
	}
	slog.Info("telegram bot authorized", slog.String("account", me.Result.Username))

	return app.NewLoop(chatModel, bot, cfg.Telegram.AdminId)
}

func initLangChain(cfg Config) (llms.ChatLLM, error) {
	switch cfg.LLMEngine {
	case "ollama":
		langChainChat, err := ollama.NewChat(
			ollama.WithLLMOptions(
				ollama.WithModel(cfg.OLLama.Model),
				ollama.WithServerURL(cfg.OLLama.Url),
			))
		if err != nil {
			return nil, fmt.Errorf("ollama chat init, %w", err)
		}
		return langChainChat, nil
	case "openai":
		langChainChat, err := openai.NewChat(
			openai.WithModel(cfg.OpenAI.Model),
			openai.WithToken(cfg.OpenAI.ApiKey),
			openai.WithHTTPClient(newOpenAiClient(cfg.OpenAI.Debug)),
		)
		if err != nil {
			return nil, fmt.Errorf("openai chat init, %w", err)
		}

		return langChainChat, nil
	}

	return nil, fmt.Errorf("unknown LLM engine %s", cfg.LLMEngine)
}

func newOpenAiClient(debug bool) *http.Client {
	if debug {
		return &http.Client{
			Transport: custom_http.DebugRoundTripper{
				Proxied: http.DefaultTransport,
			},
		}
	}

	return http.DefaultClient
}
