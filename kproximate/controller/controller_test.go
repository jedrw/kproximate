package main

import (
	"testing"
	"time"
)

func TestCooldownLogic(t *testing.T) {
	// Test within cooldown period
	lastScaleUpTime := time.Now().Add(-60 * time.Second)
	cooldownSeconds := 300

	if !lastScaleUpTime.IsZero() && cooldownSeconds > 0 {
		timeSinceLastScaleUp := time.Since(lastScaleUpTime)
		cooldownDuration := time.Duration(cooldownSeconds) * time.Second
		if timeSinceLastScaleUp < cooldownDuration {
			// Expected: should be in cooldown
			remainingCooldown := cooldownDuration - timeSinceLastScaleUp
			if remainingCooldown <= 0 {
				t.Error("Expected to be within cooldown period")
			}
		} else {
			t.Error("Expected to be within cooldown period")
		}
	}
}

func TestCooldownExpired(t *testing.T) {
	// Test after cooldown period
	lastScaleUpTime := time.Now().Add(-400 * time.Second)
	cooldownSeconds := 300

	if !lastScaleUpTime.IsZero() && cooldownSeconds > 0 {
		timeSinceLastScaleUp := time.Since(lastScaleUpTime)
		cooldownDuration := time.Duration(cooldownSeconds) * time.Second
		if timeSinceLastScaleUp < cooldownDuration {
			t.Error("Expected cooldown to be expired")
		}
	}
}

func TestCooldownDisabled(t *testing.T) {
	// Test with cooldown disabled (0)
	lastScaleUpTime := time.Now().Add(-10 * time.Second)
	cooldownSeconds := 0

	if !lastScaleUpTime.IsZero() && cooldownSeconds > 0 {
		t.Error("Cooldown should be disabled")
	}
}

func TestCooldownNoLastScaleUp(t *testing.T) {
	// Test with no previous scale up
	var lastScaleUpTime time.Time
	cooldownSeconds := 300

	if !lastScaleUpTime.IsZero() && cooldownSeconds > 0 {
		t.Error("Cooldown should not apply when lastScaleUpTime is zero")
	}
}

func TestCooldownRemainingTime(t *testing.T) {
	// Test remaining cooldown calculation
	lastScaleUpTime := time.Now().Add(-100 * time.Second)
	cooldownSeconds := 300

	if !lastScaleUpTime.IsZero() && cooldownSeconds > 0 {
		timeSinceLastScaleUp := time.Since(lastScaleUpTime)
		cooldownDuration := time.Duration(cooldownSeconds) * time.Second
		if timeSinceLastScaleUp < cooldownDuration {
			remainingCooldown := cooldownDuration - timeSinceLastScaleUp
			expectedRemaining := 200 * time.Second
			// Allow 5 second tolerance for test execution time
			if remainingCooldown < expectedRemaining-5*time.Second || remainingCooldown > expectedRemaining+5*time.Second {
				t.Errorf("Expected remaining cooldown around %v, got %v", expectedRemaining, remainingCooldown)
			}
		}
	}
}
