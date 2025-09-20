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

package v1beta1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

func TestProviderConfigDeepCopy(t *testing.T) {
	original := &ProviderConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ProviderConfig",
			APIVersion: "discord.crossplane.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-provider-config",
		},
		Spec: ProviderConfigSpec{
			Credentials: ProviderCredentials{
				Source: xpv1.CredentialsSourceSecret,
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "discord-secret",
							Namespace: "crossplane-system",
						},
						Key: "token",
					},
				},
			},
			BaseURL: stringPtr("https://discord.com/api/v10"),
		},
		Status: ProviderConfigStatus{
			ProviderConfigStatus: xpv1.ProviderConfigStatus{
				Users: 1,
			},
		},
	}

	// Test DeepCopy
	copied := original.DeepCopy()

	// Verify they're not the same object
	assert.NotSame(t, original, copied)

	// Verify the content is the same
	assert.Equal(t, original.TypeMeta, copied.TypeMeta)
	assert.Equal(t, original.ObjectMeta, copied.ObjectMeta)
	assert.Equal(t, original.Spec, copied.Spec)
	assert.Equal(t, original.Status, copied.Status)

	// Verify deep copy - modifying one shouldn't affect the other
	copied.Spec.Credentials.SecretRef.Name = "modified-secret"
	assert.NotEqual(t, original.Spec.Credentials.SecretRef.Name, copied.Spec.Credentials.SecretRef.Name)
}

func TestProviderConfigUsageDeepCopy(t *testing.T) {
	original := &ProviderConfigUsage{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ProviderConfigUsage",
			APIVersion: "discord.crossplane.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-usage",
		},
		ProviderConfigUsage: xpv1.ProviderConfigUsage{
			ProviderConfigReference: xpv1.Reference{
				Name: "test-provider-config",
			},
			ResourceReference: xpv1.TypedReference{
				APIVersion: "discord.crossplane.io/v1alpha1",
				Kind:       "Guild",
				Name:       "test-guild",
			},
		},
	}

	// Test DeepCopy
	copied := original.DeepCopy()

	// Verify they're not the same object
	assert.NotSame(t, original, copied)

	// Verify the content is the same
	assert.Equal(t, original.TypeMeta, copied.TypeMeta)
	assert.Equal(t, original.ObjectMeta, copied.ObjectMeta)
	assert.Equal(t, original.ProviderConfigUsage, copied.ProviderConfigUsage)
}

func TestProviderConfigJSONMarshaling(t *testing.T) {
	config := &ProviderConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ProviderConfig",
			APIVersion: "discord.crossplane.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: ProviderConfigSpec{
			Credentials: ProviderCredentials{
				Source: xpv1.CredentialsSourceSecret,
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "discord-creds",
							Namespace: "crossplane-system",
						},
						Key: "bot-token",
					},
				},
			},
			BaseURL: stringPtr("https://discord.com/api/v10"),
		},
	}

	// Test marshaling
	data, err := json.Marshal(config)
	require.NoError(t, err)
	assert.Contains(t, string(data), "default")
	assert.Contains(t, string(data), "discord-creds")
	assert.Contains(t, string(data), "bot-token")
	assert.Contains(t, string(data), "https://discord.com/api/v10")

	// Test unmarshaling
	var unmarshaled ProviderConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify the unmarshaled object matches
	assert.Equal(t, config.TypeMeta, unmarshaled.TypeMeta)
	assert.Equal(t, config.Name, unmarshaled.Name)
	assert.Equal(t, config.Spec.Credentials.Source, unmarshaled.Spec.Credentials.Source)
	require.NotNil(t, unmarshaled.Spec.Credentials.SecretRef)
	assert.Equal(t, "discord-creds", unmarshaled.Spec.Credentials.SecretRef.Name)
	assert.Equal(t, "bot-token", unmarshaled.Spec.Credentials.SecretRef.Key)
	require.NotNil(t, unmarshaled.Spec.BaseURL)
	assert.Equal(t, "https://discord.com/api/v10", *unmarshaled.Spec.BaseURL)
}

func TestProviderConfigUsageJSONMarshaling(t *testing.T) {
	usage := &ProviderConfigUsage{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ProviderConfigUsage",
			APIVersion: "discord.crossplane.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "guild-usage",
		},
		ProviderConfigUsage: xpv1.ProviderConfigUsage{
			ProviderConfigReference: xpv1.Reference{
				Name: "default",
			},
			ResourceReference: xpv1.TypedReference{
				APIVersion: "discord.crossplane.io/v1alpha1",
				Kind:       "Guild",
				Name:       "my-guild",
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(usage)
	require.NoError(t, err)
	assert.Contains(t, string(data), "guild-usage")
	assert.Contains(t, string(data), "default")
	assert.Contains(t, string(data), "Guild")
	assert.Contains(t, string(data), "my-guild")

	// Test unmarshaling
	var unmarshaled ProviderConfigUsage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify the unmarshaled object matches
	assert.Equal(t, usage.TypeMeta, unmarshaled.TypeMeta)
	assert.Equal(t, usage.Name, unmarshaled.Name)
	assert.Equal(t, usage.ProviderConfigReference.Name, unmarshaled.ProviderConfigReference.Name)
	assert.Equal(t, usage.ResourceReference.Kind, unmarshaled.ResourceReference.Kind)
	assert.Equal(t, usage.ResourceReference.Name, unmarshaled.ResourceReference.Name)
}

func TestProviderCredentialsValidation(t *testing.T) {
	// Test creating valid provider credentials
	creds := ProviderCredentials{
		Source: xpv1.CredentialsSourceSecret,
		CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
			SecretRef: &xpv1.SecretKeySelector{
				SecretReference: xpv1.SecretReference{
					Name:      "my-discord-secret",
					Namespace: "default",
				},
				Key: "discord-token",
			},
		},
	}

	// Verify credentials can be created and accessed
	assert.Equal(t, xpv1.CredentialsSourceSecret, creds.Source)
	require.NotNil(t, creds.SecretRef)
	assert.Equal(t, "my-discord-secret", creds.SecretRef.Name)
	assert.Equal(t, "default", creds.SecretRef.Namespace)
	assert.Equal(t, "discord-token", creds.SecretRef.Key)
}

func TestProviderConfigSpecValidation(t *testing.T) {
	// Test creating valid provider config spec with default baseURL
	spec := ProviderConfigSpec{
		Credentials: ProviderCredentials{
			Source: xpv1.CredentialsSourceSecret,
			CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
				SecretRef: &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{
						Name:      "discord-secret",
						Namespace: "crossplane-system",
					},
					Key: "token",
				},
			},
		},
		// BaseURL is optional and can be nil for default
	}

	assert.Equal(t, xpv1.CredentialsSourceSecret, spec.Credentials.Source)
	assert.Nil(t, spec.BaseURL) // Should be nil for default

	// Test with custom base URL
	customSpec := ProviderConfigSpec{
		Credentials: spec.Credentials,
		BaseURL:     stringPtr("https://discord.com/api/v9"),
	}

	require.NotNil(t, customSpec.BaseURL)
	assert.Equal(t, "https://discord.com/api/v9", *customSpec.BaseURL)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}
