package subscription

import (
	"strings"
	"testing"
)

func TestRenderUserDocumentedFormat(t *testing.T) {
	state := State{
		Version:         1,
		Name:            "olcrtc gothex",
		Refresh:         "10m",
		GeneratedAtUnix: 1778011200,
		Users: []StateUser{{
			ID:    "alice",
			Name:  "Alice",
			Token: "token",
		}},
		Entries: []Entry{
			{
				ID:           "alice-ru-telemost",
				UserID:       "alice",
				LocationName: "RU",
				LocationHost: "ru2.gothex.xyz",
				ProviderName: "Telemost",
				Auth:         "telemost",
				Transport:    "vp8channel",
				RoomID:       "35092468058950",
				Key:          strings.Repeat("a", 64),
				VP8:          &VP8{FPS: 60, BatchSize: 64},
				Comment:      "Telemost VP8",
			},
			{
				ID:           "alice-de-jitsi1",
				UserID:       "alice",
				LocationName: "DE",
				LocationHost: "de2.gothex.xyz",
				ProviderName: "Jitsi 1",
				Auth:         "jitsi",
				Transport:    "datachannel",
				RoomID:       "https://meet.playform.ru/olcrtc-alice-de-jitsi1",
				Key:          strings.Repeat("b", 64),
				Comment:      "Jitsi datachannel",
			},
		},
	}

	body, err := state.RenderToken("token")
	if err != nil {
		t.Fatalf("RenderToken() error = %v", err)
	}

	assertContains(t, body, "#name: olcrtc gothex / Alice\n")
	assertContains(t, body, "#update: 1778011200\n")
	assertContains(t, body, "#refresh: 10m\n")
	assertContains(t, body, "olcrtc://telemost?vp8channel<vp8-fps=60&vp8-batch=64>@35092468058950#"+strings.Repeat("a", 64)+"$RU / Telemost / ru2.gothex.xyz\n")
	assertContains(t, body, "olcrtc://jitsi?datachannel@https://meet.playform.ru/olcrtc-alice-de-jitsi1#"+strings.Repeat("b", 64)+"$DE / Jitsi 1 / de2.gothex.xyz\n")
	assertContains(t, body, "##name: RU / Telemost\n")
	assertContains(t, body, "##comment: Telemost VP8\n")
	if strings.Contains(body, "%") {
		t.Fatalf("rendered subscription contains legacy %% extension:\n%s", body)
	}
}

func TestRenderTokenUnknown(t *testing.T) {
	_, err := (State{}).RenderToken("missing")
	if err != ErrTokenNotFound {
		t.Fatalf("RenderToken() error = %v, want %v", err, ErrTokenNotFound)
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("rendered subscription missing %q in:\n%s", want, got)
	}
}
