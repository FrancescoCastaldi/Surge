//go:build !windows && !linux && !darwin

package power

import (
	"context"
	"testing"
)

func TestPowerOther_Controller(t *testing.T) {
	c := NewController()
	if c == nil {
		t.Fatal("expected non-nil Controller")
	}

	if err := c.Shutdown(context.Background()); err == nil {
		t.Error("expected error on Shutdown for unsupported platform, got nil")
	}

	release, err := c.AcquireInhibitor("test reason")
	if err != nil {
		t.Errorf("expected nil error on AcquireInhibitor, got %v", err)
	}
	if release == nil {
		t.Fatal("expected non-nil ReleaseFunc")
	}
	if err := release(); err != nil {
		t.Errorf("expected nil error on release, got %v", err)
	}
}
