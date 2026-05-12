package salutejazz

import (
	"context"
	"errors"
	"testing"

	"github.com/pion/webrtc/v4"
)

//nolint:cyclop // table-driven test naturally has many branches
func TestSessionStateHelpers(t *testing.T) {
	s := &Session{
		reconnectCh:    make(chan struct{}, 1),
		closeCh:        make(chan struct{}),
		sessionCloseCh: make(chan struct{}),
		sendQueue:      make(chan []byte, 1),
		subscriberConn: make(chan struct{}),
		publisherConn:  make(chan struct{}),
	}

	s.resetMediaState()
	if s.subscriberReady.Load() || s.publisherReady.Load() || s.subscriberConn == nil || s.publisherConn == nil {
		t.Fatal("resetMediaState() did not reset readiness")
	}
	if s.hasLocalVideoTracks() {
		t.Fatal("hasLocalVideoTracks() = true without tracks")
	}
	if err := s.AddVideoTrack(nil); err != nil {
		t.Fatalf("AddVideoTrack(nil) error = %v", err)
	}
	if !s.hasLocalVideoTracks() {
		t.Fatal("hasLocalVideoTracks() = false after AddVideoTrack")
	}

	s.SetVideoTrackHandler(func(*webrtc.TrackRemote, *webrtc.RTPReceiver) {})
	if s.videoTrackHandler() == nil {
		t.Fatal("videoTrackHandler() = nil")
	}

	cfg := defaultWebRTCConfig()
	if cfg.SDPSemantics != webrtc.SDPSemanticsUnifiedPlan || cfg.BundlePolicy != webrtc.BundlePolicyMaxBundle {
		t.Fatalf("defaultWebRTCConfig() = %+v", cfg)
	}
	if s.buildAPI() == nil {
		t.Fatal("buildAPI() returned nil")
	}
}

func TestSessionCallbacksQueueReconnectAndClose(t *testing.T) {
	s := &Session{
		reconnectCh:    make(chan struct{}, 1),
		closeCh:        make(chan struct{}),
		sessionCloseCh: make(chan struct{}),
		sendQueue:      make(chan []byte, 1),
	}

	s.SetReconnectCallback(func(*webrtc.DataChannel) {})
	s.SetShouldReconnect(func() bool { return true })
	s.SetEndedCallback(func(string) {})
	if s.onReconnect == nil || s.shouldReconnect == nil || s.onEnded == nil {
		t.Fatal("callbacks were not stored")
	}

	s.queueReconnect()
	select {
	case <-s.reconnectCh:
	default:
		t.Fatal("queueReconnect() did not enqueue")
	}

	s.SetShouldReconnect(func() bool { return false })
	s.queueReconnect()
	select {
	case <-s.reconnectCh:
		t.Fatal("queueReconnect() enqueued despite policy=false")
	default:
	}

	done := make(chan struct{})
	go func() {
		s.WatchConnection(context.Background())
		close(done)
	}()
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	<-done
	if err := s.Send([]byte("closed")); !errors.Is(err, ErrDataChannelNotReady) {
		t.Fatalf("Send() error = %v, want datachannel not ready", err)
	}
}

func TestSessionCanSendVideoOnlyModes(t *testing.T) {
	s := &Session{sendQueue: make(chan []byte, 1)}
	s.subscriberReady.Store(true)
	if !s.CanSend() {
		t.Fatal("CanSend() = false for subscriber-ready session without local video")
	}
	_ = s.AddVideoTrack(nil)
	if s.CanSend() {
		t.Fatal("CanSend() = true with local video but publisher not ready")
	}
	s.publisherReady.Store(true)
	if !s.CanSend() {
		t.Fatal("CanSend() = false with subscriber and publisher ready")
	}
	s.closed.Store(true)
	if s.CanSend() {
		t.Fatal("CanSend() = true for closed session")
	}
}
