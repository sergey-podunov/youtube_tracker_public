package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Notifier is a minimal interface for sending a text message using a simple HTTP POST.
// Implementations: telegram Bot API, Slack Bot DM (chat.postMessage), Slack Incoming Webhook.
// The selection is done by environment variables via NewNotifier.
// Priority: telegram > Slack Bot DM > Slack Webhook > Noop.
// If neither is configured, a Noop notifier is returned.
//
// Env vars:
// - TELEGRAM_TOKEN: telegram bot token (selects telegram if set)
// - TELEGRAM_CHAT_ID: telegram chat ID (int)
// - SLACK_WEBHOOK_URL: Slack Incoming Webhook url (used when telegram/Slack Bot not set)
//
// All implementations use a single HTTP POST per send.

type Notifier interface {
	Send(ctx context.Context, text string) error
}

type NotifierConfig struct {
	TelegramToken   string
	TelegramChatID  int64
	SlackWebhookURL string
}

type Noop struct {}

func (Noop) Send(_ context.Context, _ string) error { return nil }

type slackWebhook struct {
	url    string
	client *http.Client
}

func (s slackWebhook) Send(ctx context.Context, text string) error {
	payload := map[string]any{"text": text}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("slack webhook status %d", resp.StatusCode)
	}

	return nil
}

type tgSendPayload struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type telegram struct {
	botToken string
	chatID int64
	client *http.Client
}

func (t telegram) Send(ctx context.Context, text string) error {
	p := tgSendPayload{ChatID: t.chatID, Text: text}
	b, _ := json.Marshal(p)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer resp.Body.Close()
	
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("telegram api status %d", resp.StatusCode)
	}
	
	return nil
}

// NewNotifier chooses telegram if TELEGRAM_TOKEN is set; otherwise Slack webhook.
// If required configs are missing, falls back to Noop.
func NewNotifier(config NotifierConfig) Notifier {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	
	if len(strings.TrimSpace(config.TelegramToken)) != 0 {
		if config.TelegramChatID == 0 {
			return Noop{}
		}
		
		return telegram{
			botToken: config.TelegramToken,
			chatID:   config.TelegramChatID,
			client:   httpClient,
		}
	}
	
	if len(strings.TrimSpace(config.SlackWebhookURL)) != 0 {
		return slackWebhook{
			url:    config.SlackWebhookURL,
			client: httpClient,
		}
	}

	return Noop{}
}
