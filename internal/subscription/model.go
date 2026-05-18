// Package subscription owns olcrtc subscription state, rendering, and
// provisioning helpers.
package subscription

import "time"

const (
	DefaultRefresh = "10m"
	DefaultDNS     = "1.1.1.1:53"
)

// Desired is the hand-edited provisioning input.
type Desired struct {
	Version       int           `yaml:"version"`
	PublicBaseURL string        `yaml:"public_base_url"`
	Name          string        `yaml:"name"`
	Refresh       string        `yaml:"refresh"`
	Locations     []Location    `yaml:"locations"`
	Carriers      []Carrier     `yaml:"carriers"`
	Users         []DesiredUser `yaml:"users"`
	TelemostRooms []string      `yaml:"telemost_rooms"`
}

// DesiredUser is one subscription owner requested by the operator.
type DesiredUser struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Location is one VPS/location where srv instances run.
type Location struct {
	ID    string `yaml:"id"`
	Name  string `yaml:"name"`
	Host  string `yaml:"host"`
	IP    string `yaml:"ip"`
	Icon  string `yaml:"icon"`
	Color string `yaml:"color"`
}

// Carrier is one provider/transport option exposed in subscriptions.
type Carrier struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	Provider     string `yaml:"provider"`
	Transport    string `yaml:"transport"`
	JitsiBaseURL string `yaml:"jitsi_base_url,omitempty"`
	Icon         string `yaml:"icon"`
	Color        string `yaml:"color"`
	Comment      string `yaml:"comment"`
	VP8          *VP8   `yaml:"vp8,omitempty"`
}

// VP8 contains vp8channel tuning used in config YAML and URI payloads.
type VP8 struct {
	FPS       int `yaml:"fps"`
	BatchSize int `yaml:"batch_size"`
}

// State is the generated deployment and backend input.
type State struct {
	Version         int         `yaml:"version"`
	PublicBaseURL   string      `yaml:"public_base_url"`
	Name            string      `yaml:"name"`
	Refresh         string      `yaml:"refresh"`
	GeneratedAt     string      `yaml:"generated_at"`
	GeneratedAtUnix int64       `yaml:"generated_at_unix"`
	Users           []StateUser `yaml:"users"`
	Entries         []Entry     `yaml:"entries"`
}

// StateUser is a generated subscription owner with an opaque token.
type StateUser struct {
	ID              string `yaml:"id"`
	Name            string `yaml:"name"`
	Token           string `yaml:"token"`
	SubscriptionURL string `yaml:"subscription_url"`
}

// Entry is one olcrtc URI plus metadata and one server-side instance config.
type Entry struct {
	ID           string `yaml:"id"`
	UserID       string `yaml:"user_id"`
	UserName     string `yaml:"user_name"`
	LocationID   string `yaml:"location_id"`
	LocationName string `yaml:"location_name"`
	LocationHost string `yaml:"location_host"`
	LocationIP   string `yaml:"location_ip,omitempty"`
	ProviderID   string `yaml:"provider_id"`
	ProviderName string `yaml:"provider_name"`
	Auth         string `yaml:"auth"`
	Transport    string `yaml:"transport"`
	RoomID       string `yaml:"room_id"`
	Key          string `yaml:"key"`
	DNS          string `yaml:"dns"`
	Icon         string `yaml:"icon,omitempty"`
	Color        string `yaml:"color,omitempty"`
	Comment      string `yaml:"comment,omitempty"`
	VP8          *VP8   `yaml:"vp8,omitempty"`
}

// UserEntries returns all entries owned by userID in state order.
func (s State) UserEntries(userID string) []Entry {
	out := make([]Entry, 0)
	for _, entry := range s.Entries {
		if entry.UserID == userID {
			out = append(out, entry)
		}
	}
	return out
}

func generatedAt(now time.Time) (string, int64) {
	utc := now.UTC()
	return utc.Format(time.RFC3339), utc.Unix()
}
