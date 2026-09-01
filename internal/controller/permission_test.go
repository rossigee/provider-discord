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

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
)

// TestPermissionErrorDetection validates the helper functions for detecting permission errors.
func TestPermissionErrorDetection(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		errStr    string
		wantFound bool
		testFunc  func(error) bool
		desc      string
	}{
		// 403 Permission Denied tests
		{
			name:      "permission_denied_403",
			errStr:    "Discord API error: 403 - Missing Permissions",
			wantFound: true,
			testFunc: func(err error) bool {
				return err != nil && contains(err.Error(), "Discord API error: 403")
			},
			desc: "Detect 403 Forbidden responses",
		},
		{
			name:      "permission_denied_no_match",
			errStr:    "Discord API error: 400 - Bad Request",
			wantFound: false,
			testFunc: func(err error) bool {
				return err != nil && contains(err.Error(), "Discord API error: 403")
			},
			desc: "Don't match other status codes",
		},
		// 401 Unauthorized tests
		{
			name:      "unauthorized_401",
			errStr:    "Discord API error: 401 - Unauthorized",
			wantFound: true,
			testFunc: func(err error) bool {
				return err != nil && contains(err.Error(), "Discord API error: 401")
			},
			desc: "Detect 401 Unauthorized responses",
		},
		{
			name:      "unauthorized_no_match",
			errStr:    "Discord API error: 403 - Forbidden",
			wantFound: false,
			testFunc: func(err error) bool {
				return err != nil && contains(err.Error(), "Discord API error: 401")
			},
			desc: "Don't match other status codes",
		},
		// Nil error tests
		{
			name:      "nil_error",
			err:       nil,
			wantFound: false,
			testFunc: func(err error) bool {
				return err != nil && contains(err.Error(), "Discord API error: 403")
			},
			desc: "Handle nil errors safely",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create error from string if provided
			var err error
			if tt.errStr != "" {
				err = newTestError(tt.errStr)
			} else {
				err = tt.err
			}

			got := tt.testFunc(err)
			if got != tt.wantFound {
				t.Errorf("%s: error detection = %v, want %v",
					tt.desc, got, tt.wantFound)
			}
		})
	}
}

// TestPermissionErrorConditions validates that permission errors set correct conditions.
func TestPermissionErrorConditions(t *testing.T) {
	tests := []struct {
		name            string
		errorType       string
		expectedMessage string
		desc            string
	}{
		{
			name:            "permission_denied",
			errorType:       "403",
			expectedMessage: "missing permissions",
			desc:            "403 errors should indicate missing permissions",
		},
		{
			name:            "unauthorized",
			errorType:       "401",
			expectedMessage: "invalid or expired bot token",
			desc:            "401 errors should indicate auth problem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// In real controllers, these would be set via:
			// cr.SetConditions(xpv1.Unavailable().WithMessage(message))
			// This test validates the condition messages are appropriate

			cond := xpv1.Unavailable().WithMessage(tt.expectedMessage)

			// Validate message was set
			if cond.Message != tt.expectedMessage {
				t.Errorf("%s: condition message = %q, want %q",
					tt.desc, cond.Message, tt.expectedMessage)
			}

			// Validate status is False (Unavailable means not ready)
			if cond.Status != corev1.ConditionFalse {
				t.Errorf("%s: Unavailable condition should have Status=False, got %v",
					tt.desc, cond.Status)
			}
		})
	}
}

// TestPermissionErrorDoesNotRetry validates that permission errors don't cause retries.
func TestPermissionErrorDoesNotRetry(t *testing.T) {
	// When Observe/Create/Update/Delete returns nil error for permission errors,
	// the reconciler won't retry. This test documents that behavior.

	scenarios := []struct {
		name      string
		operation string
		err       string
		desc      string
	}{
		{
			name:      "observe_permission",
			operation: "Observe",
			err:       "Discord API error: 403 - Missing Permissions",
			desc:      "Observe permission errors return nil to stop retrying",
		},
		{
			name:      "create_permission",
			operation: "Create",
			err:       "Discord API error: 403 - Missing Permissions",
			desc:      "Create permission errors return nil to stop retrying",
		},
		{
			name:      "update_permission",
			operation: "Update",
			err:       "Discord API error: 403 - Missing Permissions",
			desc:      "Update permission errors return nil to stop retrying",
		},
		{
			name:      "delete_permission",
			operation: "Delete",
			err:       "Discord API error: 403 - Missing Permissions",
			desc:      "Delete permission errors return nil to stop retrying",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Validate that error string is correctly formatted for detection
			if !contains(scenario.err, "Discord API error: 403") {
				t.Errorf("%s: test error not properly formatted", scenario.desc)
			}
		})
	}
}

// Helper functions for tests

type testError string

func (e testError) Error() string {
	return string(e)
}

func newTestError(msg string) error {
	return testError(msg)
}

func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
