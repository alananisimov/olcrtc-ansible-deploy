package jitsi

import (
	"context"
	"errors"
	"testing"

	"github.com/openlibrecommunity/olcrtc/internal/auth"
)

func TestParseRoomURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		host    string
		room    string
		wantErr bool
	}{
		{name: "https url", raw: "https://meet.cryptopro.ru/myroom", host: "meet.cryptopro.ru", room: "myroom"},
		{name: "http url", raw: "http://meet.example/myroom", host: "meet.example", room: "myroom"},
		{name: "scheme-less", raw: "meet.example.com/myroom", host: "meet.example.com", room: "myroom"},
		{name: "trailing slash", raw: "https://meet.example/myroom/", host: "meet.example", room: "myroom"},
		{name: "double slash leader", raw: "//meet.example/myroom", host: "meet.example", room: "myroom"},
		{name: "uppercase room", raw: "https://meet.example/MyRoom", host: "meet.example", room: "MyRoom"},
		{name: "empty", raw: "", wantErr: true},
		{name: "host only", raw: "meet.example.com", wantErr: true},
		{name: "no room", raw: "https://meet.example/", wantErr: true},
		{name: "scheme only", raw: "https://", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host, room, err := parseRoomURL(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseRoomURL(%q) = (%q, %q), want error", tc.raw, host, room)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRoomURL(%q) error = %v, want nil", tc.raw, err)
			}
			if host != tc.host || room != tc.room {
				t.Fatalf("parseRoomURL(%q) = (%q, %q), want (%q, %q)",
					tc.raw, host, room, tc.host, tc.room)
			}
		})
	}
}

func TestProviderIssue(t *testing.T) {
	creds, err := Provider{}.Issue(context.Background(), auth.Config{
		RoomURL: "https://meet.cryptopro.ru/olcrtc",
		Name:    "olcrtc-test",
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if creds.URL != "meet.cryptopro.ru" {
		t.Fatalf("URL = %q, want %q", creds.URL, "meet.cryptopro.ru")
	}
	if got := creds.Extra[CredentialKeyRoom]; got != "olcrtc" {
		t.Fatalf("room = %q, want %q", got, "olcrtc")
	}
	if creds.Token != "" {
		t.Fatalf("Token = %q, want empty", creds.Token)
	}
}

func TestProviderIssueRequiresRoom(t *testing.T) {
	_, err := Provider{}.Issue(context.Background(), auth.Config{RoomURL: ""})
	if !errors.Is(err, auth.ErrRoomIDRequired) {
		t.Fatalf("Issue() err = %v, want ErrRoomIDRequired", err)
	}
}

func TestProviderEngine(t *testing.T) {
	if got := (Provider{}).Engine(); got != "jitsi" {
		t.Fatalf("Engine() = %q, want %q", got, "jitsi")
	}
	if got := (Provider{}).DefaultServiceURL(); got != "" {
		t.Fatalf("DefaultServiceURL() = %q, want empty", got)
	}
}
