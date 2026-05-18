package subscription

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesHealthAndSubscription(t *testing.T) {
	state := State{
		Version: 1,
		Name:    "test",
		Users: []StateUser{{
			ID:    "alice",
			Token: "tok",
		}},
		Entries: []Entry{{
			ID:           "alice-ru-jitsi1",
			UserID:       "alice",
			LocationName: "RU",
			ProviderName: "Jitsi 1",
			Auth:         "jitsi",
			Transport:    "datachannel",
			RoomID:       "https://meet.playform.ru/room",
			Key:          strings.Repeat("c", 64),
		}},
	}
	srv := httptest.NewServer(Handler(state))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp, err = http.Get(srv.URL + "/sub/tok") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /sub/tok: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /sub/tok status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestHandlerReturns404ForUnknownToken(t *testing.T) {
	srv := httptest.NewServer(Handler(State{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sub/missing") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /sub/missing: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
