package subscription

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrTokenNotFound = errors.New("subscription token not found")
	ErrNoEntries     = errors.New("subscription user has no entries")
)

// FindUserByToken resolves an opaque subscription token.
func (s State) FindUserByToken(token string) (StateUser, bool) {
	for _, user := range s.Users {
		if user.Token == token {
			return user, true
		}
	}
	return StateUser{}, false
}

// RenderToken renders a sub.md document for token.
func (s State) RenderToken(token string) (string, error) {
	user, ok := s.FindUserByToken(token)
	if !ok {
		return "", ErrTokenNotFound
	}
	return s.RenderUser(user)
}

// RenderUser renders a sub.md document for user.
func (s State) RenderUser(user StateUser) (string, error) {
	entries := s.UserEntries(user.ID)
	if len(entries) == 0 {
		return "", ErrNoEntries
	}

	name := s.Name
	if name == "" {
		name = "olcrtc"
	}
	if user.Name != "" {
		name += " / " + user.Name
	}
	refresh := s.Refresh
	if refresh == "" {
		refresh = DefaultRefresh
	}

	var b strings.Builder
	writeKV(&b, "#name", name)
	if s.GeneratedAtUnix > 0 {
		writeKV(&b, "#update", fmt.Sprintf("%d", s.GeneratedAtUnix))
	}
	writeKV(&b, "#refresh", refresh)
	b.WriteByte('\n')

	for i, entry := range entries {
		if i > 0 {
			b.WriteByte('\n')
		}
		uri, err := entry.URI()
		if err != nil {
			return "", err
		}
		b.WriteString(uri)
		b.WriteByte('\n')
		writeKV(&b, "##name", entry.DisplayName())
		if entry.Icon != "" {
			writeKV(&b, "##icon", entry.Icon)
		}
		if entry.Color != "" {
			writeKV(&b, "##color", entry.Color)
		}
		if entry.LocationIP != "" {
			writeKV(&b, "##ip", entry.LocationIP)
		}
		if entry.Comment != "" {
			writeKV(&b, "##comment", entry.Comment)
		}
	}

	return b.String(), nil
}

// URI renders one documented olcrtc URI. It intentionally does not emit the
// legacy "%storage-id" extension used by older clients.
func (e Entry) URI() (string, error) {
	if e.Auth == "" {
		return "", fmt.Errorf("entry %q: auth is required", e.ID)
	}
	if e.Transport == "" {
		return "", fmt.Errorf("entry %q: transport is required", e.ID)
	}
	if e.RoomID == "" {
		return "", fmt.Errorf("entry %q: room_id is required", e.ID)
	}
	if len(e.Key) != 64 {
		return "", fmt.Errorf("entry %q: key must be 64 hex characters", e.ID)
	}
	if strings.ContainsAny(e.Auth+e.Transport+e.RoomID+e.Key, "\r\n") {
		return "", fmt.Errorf("entry %q: URI fields must not contain newlines", e.ID)
	}

	return fmt.Sprintf("olcrtc://%s?%s%s@%s#%s$%s",
		e.Auth,
		e.Transport,
		e.payload(),
		e.RoomID,
		e.Key,
		cleanMeta(e.MIMO()),
	), nil
}

// DisplayName is the local subscription row name.
func (e Entry) DisplayName() string {
	parts := make([]string, 0, 2)
	if e.LocationName != "" {
		parts = append(parts, e.LocationName)
	}
	if e.ProviderName != "" {
		parts = append(parts, e.ProviderName)
	}
	if len(parts) == 0 {
		return e.ID
	}
	return strings.Join(parts, " / ")
}

// MIMO is the free-form URI comment consumed by UI clients.
func (e Entry) MIMO() string {
	parts := make([]string, 0, 3)
	if e.LocationName != "" {
		parts = append(parts, e.LocationName)
	}
	if e.ProviderName != "" {
		parts = append(parts, e.ProviderName)
	}
	if e.LocationHost != "" {
		parts = append(parts, e.LocationHost)
	}
	return strings.Join(parts, " / ")
}

func (e Entry) payload() string {
	switch e.Transport {
	case "vp8channel":
		if e.VP8 == nil {
			return ""
		}
		return fmt.Sprintf("<vp8-fps=%d&vp8-batch=%d>", e.VP8.FPS, e.VP8.BatchSize)
	default:
		return ""
	}
}

func writeKV(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(cleanMeta(value))
	b.WriteByte('\n')
}

func cleanMeta(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}
