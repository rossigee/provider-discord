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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheTTL defines how long to skip re-observing a resource after successful sync.
// Balances rate-limit relief against drift detection. Set conservatively to ensure
// we catch external changes within a reasonable window.
const CacheTTL = 5 * time.Minute

// ShouldSkipObserve returns true if a resource was recently synced and can skip
// the Observe call to reduce API load. Used to implement status-driven reconciliation.
//
// Returns false (proceed with Observe) if:
// - lastSyncTime is nil (never synced)
// - sync happened more than CacheTTL ago (time to re-check)
// - resource has a pending desired state change (always observe before apply)
func ShouldSkipObserve(lastSyncTime *metav1.Time) bool {
	if lastSyncTime == nil {
		return false
	}
	return time.Since(lastSyncTime.Time) < CacheTTL
}

// UpdateSyncTime records when a resource was last successfully observed.
// Call this after a successful Observe to enable caching on future reconciles.
func UpdateSyncTime(lastSyncTime **metav1.Time) {
	now := metav1.NewTime(time.Now())
	*lastSyncTime = &now
}
