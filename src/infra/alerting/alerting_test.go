package alerting

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"crypgo-machine/src/application/basismonitor"
)

const testToken = "123456:SECRET-TOKEN"

func TestTelegramSender_SendsHTMLMessageToTheChat(t *testing.T) {
	var path, chatID, text, parseMode string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		chatID = r.PostForm.Get("chat_id")
		text = r.PostForm.Get("text")
		parseMode = r.PostForm.Get("parse_mode")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer server.Close()

	sender, err := NewTelegramSenderWithBaseURL(server.URL, server.Client(), " "+testToken+" ", " 42 ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := sender.Send("<b>hello</b> & goodbye"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if path != "/bot"+testToken+"/sendMessage" {
		t.Errorf("unexpected request path: %s", path)
	}
	if chatID != "42" {
		t.Errorf("expected chat id 42, got %q", chatID)
	}
	if text != "<b>hello</b> & goodbye" {
		t.Errorf("unexpected message text: %q", text)
	}
	if parseMode != "HTML" {
		t.Errorf("expected parse mode HTML, got %q", parseMode)
	}
}

func TestTelegramSender_ReportsRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`))
	}))
	defer server.Close()

	sender, err := NewTelegramSenderWithBaseURL(server.URL, server.Client(), testToken, "42")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = sender.Send("hello")
	if err == nil {
		t.Fatal("expected an error when Telegram rejects the message")
	}
	if !strings.Contains(err.Error(), "chat not found") {
		t.Errorf("expected the Telegram description in the error, got %v", err)
	}
	if strings.Contains(err.Error(), testToken) {
		t.Errorf("the error must not leak the bot token: %v", err)
	}
}

func TestTelegramSender_ErrorsNeverLeakTheToken(t *testing.T) {
	t.Run("connection failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		baseURL := server.URL
		server.Close() // nothing listens on this address any more

		sender, err := NewTelegramSenderWithBaseURL(baseURL, &http.Client{Timeout: 2 * time.Second}, testToken, "42")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		sender.sleep = func(time.Duration) {}

		err = sender.Send("hello")
		if err == nil {
			t.Fatal("expected a connection error")
		}
		if strings.Contains(err.Error(), testToken) {
			t.Errorf("the error must not leak the bot token: %v", err)
		}
	})

	t.Run("unreadable body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>bad gateway</html>"))
		}))
		defer server.Close()

		sender, err := NewTelegramSenderWithBaseURL(server.URL, server.Client(), testToken, "42")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		sender.sleep = func(time.Duration) {}

		err = sender.Send("hello")
		if err == nil || !strings.Contains(err.Error(), "502") {
			t.Fatalf("expected an HTTP 502 error, got %v", err)
		}
		if strings.Contains(err.Error(), testToken) {
			t.Errorf("the error must not leak the bot token: %v", err)
		}
	})
}

func TestTelegramSender_RetriesTransientFailures(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>bad gateway</html>"))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer server.Close()

	sender, err := NewTelegramSenderWithBaseURL(server.URL, server.Client(), testToken, "42")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var pauses []time.Duration
	sender.sleep = func(d time.Duration) { pauses = append(pauses, d) }

	if err := sender.Send("hello"); err != nil {
		t.Fatalf("expected the third attempt to succeed, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 attempts, got %d", calls)
	}
	if len(pauses) != 2 || pauses[0] != telegramRetryDelay || pauses[1] != 2*telegramRetryDelay {
		t.Errorf("expected growing pauses between attempts, got %v", pauses)
	}
}

func TestTelegramSender_GivesUpAfterMaxAttempts(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests"}`))
	}))
	defer server.Close()

	sender, err := NewTelegramSenderWithBaseURL(server.URL, server.Client(), testToken, "42")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	sender.sleep = func(time.Duration) {}

	err = sender.Send("hello")
	if err == nil || !strings.Contains(err.Error(), "gave up after 3 attempts") {
		t.Fatalf("expected the sender to give up, got %v", err)
	}
	if calls != telegramMaxAttempts {
		t.Errorf("expected %d attempts, got %d", telegramMaxAttempts, calls)
	}
}

func TestTelegramSender_DoesNotRetryRejections(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":401,"description":"Unauthorized"}`))
	}))
	defer server.Close()

	sender, err := NewTelegramSenderWithBaseURL(server.URL, server.Client(), testToken, "42")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	sender.sleep = func(time.Duration) { t.Error("a rejected message must not be retried") }

	if err := sender.Send("hello"); err == nil {
		t.Fatal("expected an error for a rejected message")
	}
	if calls != 1 {
		t.Errorf("expected a single attempt, got %d", calls)
	}
}

func TestNewTelegramSender_RequiresTokenAndChat(t *testing.T) {
	if _, err := NewTelegramSender(" ", "42"); err == nil {
		t.Error("expected an error for an empty bot token")
	}
	if _, err := NewTelegramSender(testToken, ""); err == nil {
		t.Error("expected an error for an empty chat id")
	}
}

func TestFileAlertStateStore_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	store := NewFileAlertStateStore(path)

	empty, err := store.Load()
	if err != nil {
		t.Fatalf("a missing file must load as the zero state, got %v", err)
	}
	if !empty.LastAlertAt.IsZero() || empty.LastAlertBestAnnualized != 0 {
		t.Errorf("expected the zero state, got %+v", empty)
	}

	want := basismonitor.AlertState{
		LastAlertAt:             time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC),
		LastAlertBestAnnualized: 9.5,
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, err := NewFileAlertStateStore(path).Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !got.LastAlertAt.Equal(want.LastAlertAt) || got.LastAlertBestAnnualized != want.LastAlertBestAnnualized {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestFileAlertStateStore_RejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("failed to write the fixture: %v", err)
	}

	if _, err := NewFileAlertStateStore(path).Load(); err == nil {
		t.Fatal("expected an error for a corrupt state file")
	}
}

func TestMemoryAlertStateStore_KeepsStateInProcess(t *testing.T) {
	store := NewMemoryAlertStateStore()
	want := basismonitor.AlertState{LastAlertAt: time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC), LastAlertBestAnnualized: 9.5}

	if err := store.Save(want); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != want {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestWriterNotifier_PrintsInsteadOfSending(t *testing.T) {
	var out strings.Builder
	if err := NewWriterNotifier(&out).Send("the alert text"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(out.String(), "the alert text") || !strings.Contains(out.String(), "not sent") {
		t.Errorf("unexpected output: %q", out.String())
	}
}
