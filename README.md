# tg-llm-wrapper

Telegram bot wrapper for Ollama or OpenAI API.

Just a toy project to play with language models and experiment with prompts.

## Installation

Prerequisites:

- [Go](https://golang.org/doc/install)
- [Ko](https://github.com/ko-build/ko)
- [Docker](https://docs.docker.com/get-docker/) (for development)

Also, either Ollama for self-hosted language models or OpenAI token.

For development:

Ollama:

```bash
make run/local
```

OpenAI:

```bash
make run/openai
```

To deploy as a Docker container:

```bash
make deploy
```

## Configuration

See .env.dist for basic configuration.
For advanced configuration, run `make run/local --help`.
Alternatively, the following cli flags are available:

```bash
  -llm-engine string
        LLM engine to use (default "openai")
  -ollama-model string
        OLLama model to use. Choose here https://ollama.ai/library (default "openhermes")
  -ollama-url string
        OLLama url (default "http://localhost:11434")
  -openai-api-key string
        OpenAI API key
  -openai-debug
        Debug mode for OpenAI
  -openai-model string
        OpenAI model to use (default "gpt-4-1106-preview")
  -system-prompt string
        custom initial prompt
  -telegram-bot-token string
        Telegram bot token
  -telegram-debug
        Debug mode for Telegram
  -telegram-user-id int
        Telegram user id
```



## Usage

After setup, initiate a conversation with your Telegram bot to interact with the LLM wrapper.

## Acknowledgments

Kudos to the following projects:

- [Ollama](https://github.com/jmorganca/ollama)
- [langchaingo](https://github.com/tmc/langchaingo)

