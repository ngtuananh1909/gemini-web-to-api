package providers

import (
	"context"
	"sync"
	"time"
)

type StatusSnapshot struct {
	Status   string `json:"status"`
	Provider string `json:"provider"`
	Healthy  bool   `json:"healthy"`
	Models   int    `json:"models"`
	Message  string `json:"message"`
}

type RuntimeStatus struct {
	mu    sync.RWMutex
	phase string
}

func NewRuntimeStatus() *RuntimeStatus {
	return &RuntimeStatus{phase: "connecting"}
}

func (status *RuntimeStatus) Snapshot(healthy bool, modelCount int) StatusSnapshot {
	status.mu.RLock()
	phase := status.phase
	status.mu.RUnlock()
	if healthy {
		phase = "connected"
	} else if phase == "connected" {
		phase = "disconnected"
	}
	message := "Connecting to Gemini Web."
	switch phase {
	case "connected":
		message = "Gemini Web is connected."
	case "error":
		message = "Could not connect to Gemini Web. Check cookies, account slot, and network. Retrying."
	case "disconnected":
		message = "Gemini Web is disconnected. Check the session and restart if needed."
	}
	return StatusSnapshot{Status: phase, Provider: "gemini", Healthy: healthy, Models: modelCount, Message: message}
}

func (status *RuntimeStatus) setError(_ error) {
	status.mu.Lock()
	status.phase = "error"
	status.mu.Unlock()
}

func (status *RuntimeStatus) setConnected() {
	status.mu.Lock()
	status.phase = "connected"
	status.mu.Unlock()
}

func (status *RuntimeStatus) setStopped() {
	status.mu.Lock()
	status.phase = "disconnected"
	status.mu.Unlock()
}

func retryInitialConnection(ctx context.Context, init func(context.Context) error, status *RuntimeStatus, wait func(context.Context, time.Duration) bool) {
	delay := 5 * time.Second
	for ctx.Err() == nil {
		if err := init(ctx); err == nil {
			status.setConnected()
			return
		} else if ctx.Err() == nil {
			status.setError(err)
		}
		if !wait(ctx, delay) {
			return
		}
		if delay < 5*time.Minute {
			delay *= 2
			if delay > 5*time.Minute {
				delay = 5 * time.Minute
			}
		}
	}
}

func waitForInitialRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
