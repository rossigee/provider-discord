/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldSkipObserve(t *testing.T) {
	tests := []struct {
		name         string
		lastSyncTime *metav1.Time
		wantSkip     bool
		description  string
	}{
		{
			name:         "nil_timestamp",
			lastSyncTime: nil,
			wantSkip:     false,
			description:  "Never synced before - should observe",
		},
		{
			name:         "recently_synced",
			lastSyncTime: &metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
			wantSkip:     true,
			description:  "Synced 1 minute ago - still within cache window",
		},
		{
			name:         "just_synced",
			lastSyncTime: &metav1.Time{Time: time.Now()},
			wantSkip:     true,
			description:  "Just synced - definitely within cache",
		},
		{
			name:         "cache_expired",
			lastSyncTime: &metav1.Time{Time: time.Now().Add(-6 * time.Minute)},
			wantSkip:     false,
			description:  "Synced 6 minutes ago - cache expired",
		},
		{
			name:         "at_cache_boundary",
			lastSyncTime: &metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
			wantSkip:     false,
			description:  "Synced exactly 5 minutes ago - at/past boundary",
		},
		{
			name:         "near_cache_boundary",
			lastSyncTime: &metav1.Time{Time: time.Now().Add(-4*time.Minute - 59*time.Second)},
			wantSkip:     true,
			description:  "Synced 4:59 ago - just within cache window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipObserve(tt.lastSyncTime)
			if got != tt.wantSkip {
				t.Errorf("%s: ShouldSkipObserve(%v) = %v, want %v",
					tt.description, tt.lastSyncTime, got, tt.wantSkip)
			}
		})
	}
}

func TestUpdateSyncTime(t *testing.T) {
	tests := []struct {
		name            string
		beforeValue     *metav1.Time
		wantNonNil      bool
		wantWithinDelta bool
		delta           time.Duration
		description     string
	}{
		{
			name:            "update_nil_pointer",
			beforeValue:     nil,
			wantNonNil:      true,
			wantWithinDelta: true,
			delta:           1 * time.Second,
			description:     "Update nil timestamp to current time",
		},
		{
			name:            "update_old_timestamp",
			beforeValue:     &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			wantNonNil:      true,
			wantWithinDelta: true,
			delta:           1 * time.Second,
			description:     "Replace old timestamp with current time",
		},
		{
			name:            "update_recent_timestamp",
			beforeValue:     &metav1.Time{Time: time.Now().Add(-10 * time.Second)},
			wantNonNil:      true,
			wantWithinDelta: true,
			delta:           1 * time.Second,
			description:     "Update recent timestamp to now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncTime := tt.beforeValue
			beforeCheck := time.Now()
			UpdateSyncTime(&syncTime)
			afterCheck := time.Now()

			if tt.wantNonNil && syncTime == nil {
				t.Errorf("%s: UpdateSyncTime set nil, want non-nil", tt.description)
				return
			}

			if !tt.wantNonNil && syncTime != nil {
				t.Errorf("%s: UpdateSyncTime set non-nil, want nil", tt.description)
				return
			}

			if tt.wantWithinDelta && syncTime != nil {
				timeDiff := syncTime.Sub(beforeCheck)
				if timeDiff < 0 || timeDiff > tt.delta {
					t.Errorf("%s: UpdateSyncTime set to %v, expected within %v of %v",
						tt.description, syncTime, tt.delta, beforeCheck)
				}
				// Also verify it's before the "after" check
				if syncTime.After(afterCheck) {
					t.Errorf("%s: UpdateSyncTime set to future time %v (after check was %v)",
						tt.description, syncTime, afterCheck)
				}
			}
		})
	}
}

func TestCacheIntegration(t *testing.T) {
	// Test the full cache lifecycle: observe → update → skip
	var syncTime *metav1.Time

	// Step 1: First observe - no cache
	if ShouldSkipObserve(syncTime) {
		t.Error("First observe: ShouldSkipObserve should return false for nil timestamp")
	}

	// Step 2: Update sync time after successful observe
	UpdateSyncTime(&syncTime)
	if syncTime == nil {
		t.Error("After update: syncTime should not be nil")
	}

	// Step 3: Next reconcile - should skip observe
	if !ShouldSkipObserve(syncTime) {
		t.Error("Recent observe: ShouldSkipObserve should return true within cache window")
	}

	// Step 4: Old timestamp - should not skip
	oldTime := &metav1.Time{Time: time.Now().Add(-10 * time.Minute)}
	if ShouldSkipObserve(oldTime) {
		t.Error("Expired cache: ShouldSkipObserve should return false for expired timestamp")
	}
}

func BenchmarkShouldSkipObserve(b *testing.B) {
	recentTime := &metav1.Time{Time: time.Now()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShouldSkipObserve(recentTime)
	}
}

func BenchmarkUpdateSyncTime(b *testing.B) {
	var syncTime *metav1.Time

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UpdateSyncTime(&syncTime)
	}
}
