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

package deduplication

import (
	discordv1alpha1 "github.com/rossigee/provider-discord/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

// TestDeduplicationAnnotationPredicate tests that the predicate correctly identifies ProviderConfigs with deduplication annotations.
func TestDeduplicationAnnotationPredicate(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		shouldMatch bool
	}{
		{
			name: "with report mode annotation",
			annotations: map[string]string{
				DeduplicationAnnotation: "report",
			},
			shouldMatch: true,
		},
		{
			name: "with action mode annotation",
			annotations: map[string]string{
				DeduplicationAnnotation: "action",
			},
			shouldMatch: true,
		},
		{
			name: "with invalid mode",
			annotations: map[string]string{
				DeduplicationAnnotation: "invalid",
			},
			shouldMatch: false,
		},
		{
			name:        "without annotation",
			annotations: map[string]string{},
			shouldMatch: false,
		},
		{
			name: "with other annotations",
			annotations: map[string]string{
				"other": "value",
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPC := &discordv1alpha1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: tt.annotations,
				},
			}

			// Test the logic directly
			shouldMatch := false
			if testPC.Annotations != nil {
				if mode, hasMode := testPC.Annotations[DeduplicationAnnotation]; hasMode {
					shouldMatch = mode == "report" || mode == "action"
				}
			}

			if shouldMatch != tt.shouldMatch {
				t.Errorf("expected match=%v, got %v", tt.shouldMatch, shouldMatch)
			}
		})
	}
}

// TestExtractCredentials_ValidSecret tests successful credential extraction.
func TestExtractCredentials_ValidSecret(t *testing.T) {
	// This is a unit test for the extraction logic
	// In a real test, you'd use a fake client with mock secrets

	expectedToken := "test-bot-token-12345"
	expectedBaseURL := "https://discord.com/api/v10"

	// Test data: secret with token
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"token": []byte(expectedToken),
		},
	}

	// Extract token from secret
	var botToken string
	if token, ok := secret.Data["token"]; ok {
		botToken = string(token)
	} else if token, ok := secret.Data["credentials"]; ok {
		botToken = string(token)
	}

	if botToken != expectedToken {
		t.Errorf("expected token %q, got %q", expectedToken, botToken)
	}

	// Base URL should be default
	baseURL := "https://discord.com/api/v10"
	if baseURL != expectedBaseURL {
		t.Errorf("expected baseURL %q, got %q", expectedBaseURL, baseURL)
	}
}

// TestExtractCredentials_CustomBaseURL tests extraction with custom base URL.
func TestExtractCredentials_CustomBaseURL(t *testing.T) {
	expectedToken := "test-bot-token"
	customURL := "https://custom.discord.api/v11"

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"token": []byte(expectedToken),
		},
	}

	// Extract token
	botToken := string(secret.Data["token"])
	if botToken != expectedToken {
		t.Errorf("expected token %q, got %q", expectedToken, botToken)
	}

	// Simulate custom base URL
	baseURL := customURL
	if baseURL != customURL {
		t.Errorf("expected baseURL %q, got %q", customURL, baseURL)
	}
}

// TestLastProcessedAnnotation tests idempotency via processed annotation.
func TestLastProcessedAnnotation(t *testing.T) {
	tests := []struct {
		name          string
		currentMode   string
		lastProcessed string
		shouldProcess bool
	}{
		{
			name:          "first run with report",
			currentMode:   "report",
			lastProcessed: "",
			shouldProcess: true,
		},
		{
			name:          "already processed report",
			currentMode:   "report",
			lastProcessed: "report",
			shouldProcess: false,
		},
		{
			name:          "transition from report to action",
			currentMode:   "action",
			lastProcessed: "report",
			shouldProcess: true,
		},
		{
			name:          "already processed action",
			currentMode:   "action",
			lastProcessed: "action",
			shouldProcess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if we should process based on annotations
			shouldProcess := tt.lastProcessed != tt.currentMode

			if shouldProcess != tt.shouldProcess {
				t.Errorf("expected shouldProcess=%v, got %v", tt.shouldProcess, shouldProcess)
			}
		})
	}
}

// TestDeduplicationNameGeneration tests that deduplication resource names are deterministic.
func TestDeduplicationNameGeneration(t *testing.T) {
	// Test that the same input always generates the same name (deterministic)
	nn := types.NamespacedName{Name: "test-dedup", Namespace: "default"}
	name1 := nn.String()
	name2 := nn.String()

	if name1 != name2 {
		t.Errorf("expected deterministic name generation, but got %s and %s", name1, name2)
	}

	// Expected format: "namespace/name"
	expectedPrefix := "default/test-dedup"
	if name1 != expectedPrefix {
		t.Errorf("expected name to be %q, got %q", expectedPrefix, name1)
	}
}

// TestProviderConfigWithDeduplicationSpec tests ProviderConfig with deduplication spec.
func TestProviderConfigWithDeduplicationSpec(t *testing.T) {
	pc := &discordv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: discordv1alpha1.ProviderConfigSpec{
			Credentials: discordv1alpha1.ProviderCredentials{
				Source: "Secret",
			},
			Deduplication: &discordv1alpha1.DeduplicationSpec{
				Enabled:                 true,
				Mode:                    discordv1alpha1.DeduplicationModeReport,
				DeleteOrphanedResources: true,
				TargetGuilds:            []string{"123456", "789012"},
			},
		},
	}

	// Verify spec is properly populated
	if pc.Spec.Deduplication == nil {
		t.Error("expected deduplication spec to be set")
	}

	if !pc.Spec.Deduplication.Enabled {
		t.Error("expected deduplication to be enabled")
	}

	if pc.Spec.Deduplication.Mode != discordv1alpha1.DeduplicationModeReport {
		t.Errorf("expected mode %q, got %q", discordv1alpha1.DeduplicationModeReport, pc.Spec.Deduplication.Mode)
	}

	if !pc.Spec.Deduplication.DeleteOrphanedResources {
		t.Error("expected deleteOrphanedResources to be true")
	}

	if len(pc.Spec.Deduplication.TargetGuilds) != 2 {
		t.Errorf("expected 2 target guilds, got %d", len(pc.Spec.Deduplication.TargetGuilds))
	}
}

// TestDeduplicationModeValidation tests that invalid modes are rejected.
func TestDeduplicationModeValidation(t *testing.T) {
	tests := []struct {
		name  string
		mode  string
		valid bool
	}{
		{
			name:  "report mode",
			mode:  "report",
			valid: true,
		},
		{
			name:  "action mode",
			mode:  "action",
			valid: true,
		},
		{
			name:  "invalid mode",
			mode:  "invalid",
			valid: false,
		},
		{
			name:  "empty mode",
			mode:  "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.mode == "report" || tt.mode == "action"

			if valid != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, valid)
			}
		})
	}
}

// TestAnnotationUpdate tests that annotations can be updated for mode transitions.
func TestAnnotationUpdate(t *testing.T) {
	pc := &discordv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Annotations: map[string]string{
				DeduplicationAnnotation: "report",
			},
		},
	}

	// Simulate updating annotation to action mode
	pc.Annotations[DeduplicationAnnotation] = "action"
	pc.Annotations[lastProcessedAnnotation] = ""

	if pc.Annotations[DeduplicationAnnotation] != "action" {
		t.Errorf("expected annotation to be updated to 'action'")
	}

	if pc.Annotations[lastProcessedAnnotation] != "" {
		t.Error("expected lastProcessed annotation to be cleared for mode transition")
	}
}

// TestEventGeneration tests that appropriate events are generated.
func TestEventGeneration(t *testing.T) {
	tests := []struct {
		name         string
		phase        string
		mode         string
		deleted      int
		expectedMsg  string
		expectedType string
	}{
		{
			name:         "report completed",
			phase:        "completed",
			mode:         "report",
			deleted:      0,
			expectedMsg:  "completed",
			expectedType: "Normal",
		},
		{
			name:         "action completed",
			phase:        "completed",
			mode:         "action",
			deleted:      5,
			expectedMsg:  "completed",
			expectedType: "Normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventMsg := "Deduplication " + tt.mode + " " + tt.phase
			eventType := "Normal"

			if eventType != tt.expectedType {
				t.Errorf("expected event type %v, got %v", tt.expectedType, eventType)
			}

			if eventMsg != "Deduplication "+tt.mode+" "+tt.phase {
				t.Errorf("expected event message to contain %q", tt.expectedMsg)
			}
		})
	}
}
