package scheduler

import (
	"sync"
	"testing"
)

// These tests exercise the scheduler's state machine without calling Start,
// which validates data sources over the network.

func TestSentimentScheduler_IsNotRunningUntilStartFinishes(t *testing.T) {
	s := NewSentimentScheduler(nil, nil, nil)

	if s.IsRunning() {
		t.Fatal("a new scheduler must not be running")
	}

	if err := s.beginStart(); err != nil {
		t.Fatalf("beginStart failed: %v", err)
	}
	if s.IsRunning() {
		t.Error("scheduler must not report running while data sources are still being validated")
	}

	if err := s.beginStart(); err == nil {
		t.Error("a second start while one is in progress must be rejected")
	}

	stop := s.finishStart(true)
	if stop == nil {
		t.Fatal("expected a stop channel after a successful start")
	}
	if !s.IsRunning() {
		t.Error("scheduler must report running after a successful start")
	}
	if err := s.beginStart(); err == nil {
		t.Error("starting a running scheduler must be rejected")
	}
}

func TestSentimentScheduler_FailedStartLeavesItStopped(t *testing.T) {
	s := NewSentimentScheduler(nil, nil, nil)

	if err := s.beginStart(); err != nil {
		t.Fatalf("beginStart failed: %v", err)
	}
	if stop := s.finishStart(false); stop != nil {
		t.Error("a failed start must not return a stop channel")
	}
	if s.IsRunning() {
		t.Error("scheduler must not report running after a failed start")
	}
	if err := s.TriggerManualAnalysis(); err == nil {
		t.Error("manual trigger must be rejected while the scheduler is stopped")
	}
	if err := s.beginStart(); err != nil {
		t.Errorf("a new start must be allowed after a failed one, got %v", err)
	}
}

func TestSentimentScheduler_StopClosesChannelAndAllowsRestart(t *testing.T) {
	s := NewSentimentScheduler(nil, nil, nil)

	// Stopping a scheduler that never started must be a no-op.
	s.Stop()

	if err := s.beginStart(); err != nil {
		t.Fatalf("beginStart failed: %v", err)
	}
	first := s.finishStart(true)

	s.Stop()
	if s.IsRunning() {
		t.Error("scheduler must not report running after Stop")
	}
	select {
	case <-first:
	default:
		t.Error("Stop must close the stop channel so the worker goroutines exit")
	}

	// A second Stop must not panic on the already closed channel.
	s.Stop()

	if err := s.beginStart(); err != nil {
		t.Fatalf("restart after Stop must be allowed, got %v", err)
	}
	second := s.finishStart(true)
	select {
	case <-second:
		t.Error("a restarted scheduler must get a fresh, open stop channel")
	default:
	}
}

// Run with -race: Start runs in its own goroutine while HTTP handlers read the state.
func TestSentimentScheduler_ConcurrentStateAccess(t *testing.T) {
	s := NewSentimentScheduler(nil, nil, nil)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = s.IsRunning()
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 200; j++ {
			if err := s.beginStart(); err != nil {
				continue
			}
			s.finishStart(j%2 == 0)
			s.Stop()
		}
	}()

	wg.Wait()

	if s.IsRunning() {
		t.Error("scheduler must end stopped")
	}
}
