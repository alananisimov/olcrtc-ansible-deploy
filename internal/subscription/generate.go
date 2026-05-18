package subscription

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var (
	ErrDuplicateUserID     = errors.New("duplicate user id")
	ErrDuplicateLocationID = errors.New("duplicate location id")
	ErrDuplicateCarrierID  = errors.New("duplicate carrier id")
	ErrDuplicateRoom       = errors.New("duplicate generated room")
	ErrTelemostPoolEmpty   = errors.New("telemost room pool exhausted")
	ErrInvalidID           = errors.New("id must match [a-z0-9][a-z0-9_-]*")
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// GenerateOptions controls non-deterministic generation. Tests can inject
// fixed time and entropy.
type GenerateOptions struct {
	Now  time.Time
	Rand io.Reader
}

// Generate creates or updates generated state from desired input while
// preserving existing tokens, keys, and rooms for unchanged entry IDs.
func Generate(desired Desired, previous State, opts GenerateOptions) (State, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.Rand == nil {
		opts.Rand = rand.Reader
	}
	if desired.Refresh == "" {
		desired.Refresh = DefaultRefresh
	}
	desired.PublicBaseURL = strings.TrimRight(strings.TrimSpace(desired.PublicBaseURL), "/")

	if err := validateDesired(desired); err != nil {
		return State{}, err
	}

	usersByID := make(map[string]StateUser, len(previous.Users))
	for _, user := range previous.Users {
		usersByID[user.ID] = user
	}
	entriesByID := make(map[string]Entry, len(previous.Entries))
	for _, entry := range previous.Entries {
		entriesByID[entry.ID] = entry
	}

	state := State{
		Version:       1,
		PublicBaseURL: desired.PublicBaseURL,
		Name:          desired.Name,
		Refresh:       desired.Refresh,
	}
	if previous.GeneratedAt != "" || previous.GeneratedAtUnix != 0 {
		state.GeneratedAt = previous.GeneratedAt
		state.GeneratedAtUnix = previous.GeneratedAtUnix
	} else {
		state.GeneratedAt, state.GeneratedAtUnix = generatedAt(opts.Now)
	}

	usedRooms := make(map[string]string)
	usedTelemostRooms := make(map[string]struct{})
	for _, userSpec := range desired.Users {
		for _, location := range desired.Locations {
			for _, carrier := range desired.Carriers {
				if carrier.Provider != "telemost" {
					continue
				}
				entryID := strings.Join([]string{userSpec.ID, location.ID, carrier.ID}, "-")
				if prev, ok := entriesByID[entryID]; ok && prev.RoomID != "" {
					usedTelemostRooms[prev.RoomID] = struct{}{}
				}
			}
		}
	}

	for _, userSpec := range desired.Users {
		user := StateUser{
			ID:   userSpec.ID,
			Name: userSpec.Name,
		}
		if user.Name == "" {
			user.Name = user.ID
		}
		if prev, ok := usersByID[user.ID]; ok && prev.Token != "" {
			user.Token = prev.Token
		} else {
			token, err := randomHex(opts.Rand, 16)
			if err != nil {
				return State{}, fmt.Errorf("generate token for user %q: %w", user.ID, err)
			}
			user.Token = token
		}
		if desired.PublicBaseURL != "" {
			user.SubscriptionURL = desired.PublicBaseURL + "/sub/" + user.Token
		}
		state.Users = append(state.Users, user)

		for _, location := range desired.Locations {
			for _, carrier := range desired.Carriers {
				entryID := strings.Join([]string{user.ID, location.ID, carrier.ID}, "-")
				entry := Entry{
					ID:           entryID,
					UserID:       user.ID,
					UserName:     user.Name,
					LocationID:   location.ID,
					LocationName: location.Name,
					LocationHost: location.Host,
					LocationIP:   location.IP,
					ProviderID:   carrier.ID,
					ProviderName: carrier.Name,
					Auth:         carrier.Provider,
					Transport:    carrier.Transport,
					DNS:          DefaultDNS,
					Icon:         firstNonEmpty(carrier.Icon, location.Icon),
					Color:        firstNonEmpty(carrier.Color, location.Color),
					Comment:      firstNonEmpty(carrier.Comment, "srv: "+location.Host),
					VP8:          cloneVP8(carrier.VP8),
				}

				if prev, ok := entriesByID[entryID]; ok && reusableEntry(prev, carrier) {
					entry.RoomID = prev.RoomID
					entry.Key = prev.Key
				} else {
					key, err := randomHex(opts.Rand, 32)
					if err != nil {
						return State{}, fmt.Errorf("generate key for entry %q: %w", entryID, err)
					}
					entry.Key = key
					room, err := allocateRoom(opts.Rand, carrier, user, location, desired.TelemostRooms, usedTelemostRooms)
					if err != nil {
						return State{}, fmt.Errorf("entry %q: %w", entryID, err)
					}
					entry.RoomID = room
				}

				if carrier.Provider == "telemost" {
					usedTelemostRooms[entry.RoomID] = struct{}{}
				}
				roomKey := entry.Auth + "|" + entry.RoomID
				if owner, exists := usedRooms[roomKey]; exists {
					return State{}, fmt.Errorf("%w: %s used by %s and %s", ErrDuplicateRoom, entry.RoomID, owner, entry.ID)
				}
				usedRooms[roomKey] = entry.ID
				state.Entries = append(state.Entries, entry)
			}
		}
	}

	if materialEqual(state, previous) {
		state.GeneratedAt = previous.GeneratedAt
		state.GeneratedAtUnix = previous.GeneratedAtUnix
	} else {
		state.GeneratedAt, state.GeneratedAtUnix = generatedAt(opts.Now)
	}

	return state, nil
}

func validateDesired(desired Desired) error {
	if desired.PublicBaseURL == "" {
		return errors.New("public_base_url is required")
	}
	if len(desired.Locations) == 0 {
		return errors.New("at least one location is required")
	}
	if len(desired.Carriers) == 0 {
		return errors.New("at least one carrier is required")
	}
	seenUsers := make(map[string]struct{}, len(desired.Users))
	for _, user := range desired.Users {
		if !idPattern.MatchString(user.ID) {
			return fmt.Errorf("user %q: %w", user.ID, ErrInvalidID)
		}
		if _, ok := seenUsers[user.ID]; ok {
			return fmt.Errorf("%w: %s", ErrDuplicateUserID, user.ID)
		}
		seenUsers[user.ID] = struct{}{}
	}
	seenLocations := make(map[string]struct{}, len(desired.Locations))
	for _, location := range desired.Locations {
		if !idPattern.MatchString(location.ID) {
			return fmt.Errorf("location %q: %w", location.ID, ErrInvalidID)
		}
		if location.Name == "" || location.Host == "" {
			return fmt.Errorf("location %q: name and host are required", location.ID)
		}
		if _, ok := seenLocations[location.ID]; ok {
			return fmt.Errorf("%w: %s", ErrDuplicateLocationID, location.ID)
		}
		seenLocations[location.ID] = struct{}{}
	}
	seenCarriers := make(map[string]struct{}, len(desired.Carriers))
	for _, carrier := range desired.Carriers {
		if !idPattern.MatchString(carrier.ID) {
			return fmt.Errorf("carrier %q: %w", carrier.ID, ErrInvalidID)
		}
		if carrier.Name == "" || carrier.Provider == "" || carrier.Transport == "" {
			return fmt.Errorf("carrier %q: name, provider, and transport are required", carrier.ID)
		}
		if carrier.Provider == "jitsi" && strings.TrimSpace(carrier.JitsiBaseURL) == "" {
			return fmt.Errorf("carrier %q: jitsi_base_url is required", carrier.ID)
		}
		if carrier.Provider == "telemost" && len(desired.TelemostRooms) == 0 {
			return fmt.Errorf("carrier %q: telemost_rooms are required", carrier.ID)
		}
		if _, ok := seenCarriers[carrier.ID]; ok {
			return fmt.Errorf("%w: %s", ErrDuplicateCarrierID, carrier.ID)
		}
		seenCarriers[carrier.ID] = struct{}{}
	}
	return nil
}

func allocateRoom(
	r io.Reader,
	carrier Carrier,
	user StateUser,
	location Location,
	telemostRooms []string,
	usedTelemostRooms map[string]struct{},
) (string, error) {
	switch carrier.Provider {
	case "telemost":
		for _, room := range telemostRooms {
			room = strings.TrimSpace(room)
			if room == "" {
				continue
			}
			if _, used := usedTelemostRooms[room]; used {
				continue
			}
			usedTelemostRooms[room] = struct{}{}
			return room, nil
		}
		return "", ErrTelemostPoolEmpty
	case "jitsi":
		suffix, err := randomHex(r, 4)
		if err != nil {
			return "", err
		}
		roomName := strings.Join([]string{"olcrtc", user.ID, location.ID, carrier.ID, suffix}, "-")
		return strings.TrimRight(carrier.JitsiBaseURL, "/") + "/" + roomName, nil
	default:
		return "", fmt.Errorf("unsupported provider %q", carrier.Provider)
	}
}

func reusableEntry(prev Entry, carrier Carrier) bool {
	if prev.Auth != carrier.Provider || prev.Transport != carrier.Transport || prev.RoomID == "" || prev.Key == "" {
		return false
	}
	if carrier.Provider == "jitsi" {
		return strings.HasPrefix(prev.RoomID, strings.TrimRight(carrier.JitsiBaseURL, "/")+"/")
	}
	return true
}

func randomHex(r io.Reader, bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func cloneVP8(in *VP8) *VP8 {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func materialEqual(a, b State) bool {
	a.GeneratedAt = ""
	a.GeneratedAtUnix = 0
	b.GeneratedAt = ""
	b.GeneratedAtUnix = 0
	return reflect.DeepEqual(a, b)
}
