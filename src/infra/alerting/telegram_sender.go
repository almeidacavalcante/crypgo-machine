// Package alerting holds the small, dependency-free adapters used by
// scheduled jobs to deliver alerts and remember what was already sent.
package alerting

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTelegramBaseURL = "https://api.telegram.org"
	telegramTimeout        = 20 * time.Second
	telegramMaxBodyBytes   = 1 << 20

	// Transient failures (network errors, HTTP 429 and 5xx) are retried, with
	// a pause of retryDelay times the attempt number between tries.
	telegramMaxAttempts = 3
	telegramRetryDelay  = 2 * time.Second
)

// TelegramSender sends HTML messages to one chat through the Telegram Bot
// API using only the standard library.
//
// The bot token is part of the request URL, so errors returned by this type
// never include the URL: job logs may be public.
type TelegramSender struct {
	httpClient *http.Client
	baseURL    string
	botToken   string
	chatID     string
	retryDelay time.Duration
	sleep      func(time.Duration)
}

func NewTelegramSender(botToken, chatID string) (*TelegramSender, error) {
	return NewTelegramSenderWithBaseURL(defaultTelegramBaseURL, &http.Client{Timeout: telegramTimeout}, botToken, chatID)
}

// NewTelegramSenderWithBaseURL builds a sender against another host, such as
// a test server.
func NewTelegramSenderWithBaseURL(baseURL string, httpClient *http.Client, botToken, chatID string) (*TelegramSender, error) {
	if strings.TrimSpace(botToken) == "" {
		return nil, errors.New("telegram bot token cannot be empty")
	}
	if strings.TrimSpace(chatID) == "" {
		return nil, errors.New("telegram chat id cannot be empty")
	}
	return &TelegramSender{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		botToken:   strings.TrimSpace(botToken),
		chatID:     strings.TrimSpace(chatID),
		retryDelay: telegramRetryDelay,
		sleep:      time.Sleep,
	}, nil
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

// Send delivers the message using Telegram HTML formatting, retrying
// transient failures. A timed-out request may still have been delivered, so a
// retry can occasionally produce a duplicate message.
func (s *TelegramSender) Send(message string) error {
	var err error
	for attempt := 1; attempt <= telegramMaxAttempts; attempt++ {
		var retryable bool
		retryable, err = s.sendOnce(message)
		if err == nil {
			return nil
		}
		if !retryable {
			return err
		}
		if attempt < telegramMaxAttempts {
			s.sleep(s.retryDelay * time.Duration(attempt))
		}
	}
	return fmt.Errorf("gave up after %d attempts: %w", telegramMaxAttempts, err)
}

// sendOnce makes one delivery attempt and reports whether a failure is worth
// retrying.
func (s *TelegramSender) sendOnce(message string) (retryable bool, err error) {
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", s.baseURL, s.botToken)
	form := url.Values{
		"chat_id":                  {s.chatID},
		"text":                     {message},
		"parse_mode":               {"HTML"},
		"disable_web_page_preview": {"true"},
	}

	resp, err := s.httpClient.PostForm(endpoint, form)
	if err != nil {
		return true, fmt.Errorf("telegram request failed: %w", withoutURL(err))
	}
	defer resp.Body.Close()

	transient := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError

	body, err := io.ReadAll(io.LimitReader(resp.Body, telegramMaxBodyBytes))
	if err != nil {
		return true, fmt.Errorf("failed to read telegram response: %w", err)
	}

	var parsed telegramResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return transient, fmt.Errorf("telegram returned HTTP %d with an unreadable body", resp.StatusCode)
	}
	if !parsed.OK {
		return transient, fmt.Errorf("telegram rejected the message: %d %s", parsed.ErrorCode, parsed.Description)
	}
	return false, nil
}

// withoutURL strips the request URL (which contains the bot token) from
// errors produced by net/http.
func withoutURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
