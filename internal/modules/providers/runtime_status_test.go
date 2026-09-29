package providers

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRuntimeStatusTracksReadinessWithoutExposingErrors(t *testing.T) {
	status := NewRuntimeStatus()
	if snapshot := status.Snapshot(false, 0); snapshot.Status != "connecting" || snapshot.Healthy {
		t.Fatalf("initial snapshot = %#v", snapshot)
	}
	status.setError(errors.New("secret upstream response"))
	if snapshot := status.Snapshot(false, 0); snapshot.Status != "error" || snapshot.Message == "secret upstream response" {
		t.Fatalf("error snapshot = %#v", snapshot)
	}
	status.setConnected()
	if snapshot := status.Snapshot(true, 3); snapshot.Status != "connected" || snapshot.Models != 3 {
		t.Fatalf("connected snapshot = %#v", snapshot)
	}
	if snapshot := status.Snapshot(false, 3); snapshot.Status != "disconnected" {
		t.Fatalf("unhealthy snapshot = %#v", snapshot)
	}
}

func TestRetryInitialConnectionUsesBoundedBackoff(t *testing.T) {
	status := NewRuntimeStatus()
	attempts := 0
	var delays []time.Duration
	init := func(context.Context) error {
		attempts++
		if attempts < 10 {
			return errors.New("temporary failure")
		}
		return nil
	}
	wait := func(_ context.Context, delay time.Duration) bool {
		delays = append(delays, delay)
		return true
	}
	retryInitialConnection(context.Background(), init, status, wait)
	if attempts != 10 {
		t.Errorf("attempts = %d, want 10", attempts)
	}
	want := []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second, 80 * time.Second, 160 * time.Second, 300 * time.Second, 300 * time.Second, 300 * time.Second}
	if !reflect.DeepEqual(delays, want) {
		t.Errorf("delays = %v, want %v", delays, want)
	}
	if snapshot := status.Snapshot(true, 1); snapshot.Status != "connected" {
		t.Errorf("snapshot = %#v, want connected", snapshot)
	}
}

func TestRetryInitialConnectionStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	status := NewRuntimeStatus()
	attempts := 0
	init := func(context.Context) error {
		attempts++
		return errors.New("unavailable")
	}
	wait := func(ctx context.Context, _ time.Duration) bool {
		cancel()
		<-ctx.Done()
		return false
	}
	retryInitialConnection(ctx, init, status, wait)
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}
