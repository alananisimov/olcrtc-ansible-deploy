package subscription

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGenerateCreatesAllLocationCarrierEntriesAndIsIdempotent(t *testing.T) {
	desired := testDesired()
	var entropy countingReader
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	state, err := Generate(desired, State{}, GenerateOptions{Now: now, Rand: &entropy})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(state.Users) != 1 {
		t.Fatalf("users = %d, want 1", len(state.Users))
	}
	if len(state.Entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(state.Entries))
	}
	if state.Users[0].SubscriptionURL == "" {
		t.Fatal("subscription URL is empty")
	}
	telemostRooms := make(map[string]struct{})
	for _, entry := range state.Entries {
		if len(entry.Key) != 64 {
			t.Fatalf("entry %s key length = %d", entry.ID, len(entry.Key))
		}
		if entry.Auth == "telemost" {
			telemostRooms[entry.RoomID] = struct{}{}
			if entry.Transport != "vp8channel" || entry.VP8 == nil || entry.VP8.FPS != 60 || entry.VP8.BatchSize != 64 {
				t.Fatalf("bad telemost entry: %+v", entry)
			}
		}
		if entry.Auth == "jitsi" && !strings.HasPrefix(entry.RoomID, "https://meet.playform.ru/") {
			t.Fatalf("jitsi room = %q", entry.RoomID)
		}
	}
	if len(telemostRooms) != 2 {
		t.Fatalf("telemost unique rooms = %d, want 2", len(telemostRooms))
	}

	var otherEntropy countingReader = 200
	again, err := Generate(desired, state, GenerateOptions{
		Now:  now.Add(time.Hour),
		Rand: &otherEntropy,
	})
	if err != nil {
		t.Fatalf("Generate() second run error = %v", err)
	}
	if !reflect.DeepEqual(state, again) {
		t.Fatalf("second generate changed state\nfirst:  %+v\nsecond: %+v", state, again)
	}
}

func TestGenerateRejectsDuplicateUsers(t *testing.T) {
	desired := testDesired()
	desired.Users = append(desired.Users, DesiredUser{ID: "alice"})

	_, err := Generate(desired, State{}, GenerateOptions{Rand: new(countingReader)})
	if !errors.Is(err, ErrDuplicateUserID) {
		t.Fatalf("Generate() error = %v, want %v", err, ErrDuplicateUserID)
	}
}

func TestGenerateDetectsDuplicateRoomsFromPreviousState(t *testing.T) {
	desired := testDesired()
	previous := State{
		Version: 1,
		Users: []StateUser{{
			ID:    "alice",
			Token: "tok",
		}},
		Entries: []Entry{
			{ID: "alice-ru-telemost", Auth: "telemost", Transport: "vp8channel", RoomID: "same", Key: strings.Repeat("a", 64)},
			{ID: "alice-de-telemost", Auth: "telemost", Transport: "vp8channel", RoomID: "same", Key: strings.Repeat("b", 64)},
		},
	}

	_, err := Generate(desired, previous, GenerateOptions{Rand: new(countingReader)})
	if !errors.Is(err, ErrDuplicateRoom) {
		t.Fatalf("Generate() error = %v, want %v", err, ErrDuplicateRoom)
	}
}

func TestGenerateFailsWhenTelemostPoolIsExhausted(t *testing.T) {
	desired := testDesired()
	desired.TelemostRooms = []string{"only-one"}

	_, err := Generate(desired, State{}, GenerateOptions{Rand: new(countingReader)})
	if !errors.Is(err, ErrTelemostPoolEmpty) {
		t.Fatalf("Generate() error = %v, want %v", err, ErrTelemostPoolEmpty)
	}
}

func testDesired() Desired {
	return Desired{
		Version:       1,
		PublicBaseURL: "https://olcrtc.gothex.xyz",
		Name:          "test",
		Refresh:       "10m",
		Users: []DesiredUser{{
			ID:   "alice",
			Name: "Alice",
		}},
		Locations: []Location{
			{ID: "ru", Name: "RU", Host: "ru2.gothex.xyz"},
			{ID: "de", Name: "DE", Host: "de2.gothex.xyz"},
		},
		Carriers: []Carrier{
			{
				ID:        "telemost",
				Name:      "Telemost",
				Provider:  "telemost",
				Transport: "vp8channel",
				VP8:       &VP8{FPS: 60, BatchSize: 64},
			},
			{
				ID:           "jitsi1",
				Name:         "Jitsi 1",
				Provider:     "jitsi",
				Transport:    "datachannel",
				JitsiBaseURL: "https://meet.playform.ru",
			},
		},
		TelemostRooms: []string{"room-1", "room-2"},
	}
}

type countingReader byte

func (r *countingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(*r)
		*r = *r + 1
	}
	return len(p), nil
}
