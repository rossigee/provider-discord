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

package integration

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	"github.com/rossigee/provider-discord/apis"
	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	guildv1alpha1 "github.com/rossigee/provider-discord/apis/guild/v1alpha1"
	"github.com/rossigee/provider-discord/apis/v1beta1"
)

func TestGuildLifecycle(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = apis.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	ctx := context.Background()

	// Test Guild creation
	guild := &guildv1alpha1.Guild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-guild",
			Namespace: "default",
		},
		Spec: guildv1alpha1.GuildSpec{
			ForProvider: guildv1alpha1.GuildParameters{
				Name:                        "Test Guild",
				VerificationLevel:           func() *int { v := 1; return &v }(),
				DefaultMessageNotifications: func() *int { v := 0; return &v }(),
			},
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{
					Name: "test-provider-config",
				},
			},
		},
	}

	// Create the guild resource
	if err := fakeClient.Create(ctx, guild); err != nil {
		t.Fatalf("Failed to create guild: %v", err)
	}

	// Verify it was created
	var createdGuild guildv1alpha1.Guild
	if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(guild), &createdGuild); err != nil {
		t.Fatalf("Failed to get created guild: %v", err)
	}

	// Test setting external name (simulating controller behavior)
	meta.SetExternalName(&createdGuild, "123456789")
	if err := fakeClient.Update(ctx, &createdGuild); err != nil {
		t.Fatalf("Failed to update guild with external name: %v", err)
	}

	// Test status update (simulating controller behavior)
	// For integration tests, we'll just update the object directly
	createdGuild.Status.AtProvider = guildv1alpha1.GuildObservation{
		ID:      "123456789",
		Name:    "Test Guild",
		OwnerID: "987654321",
	}
	createdGuild.Status.Conditions = []xpv1.Condition{
		{
			Type:               xpv1.TypeReady,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Reason:             "Available",
		},
	}
	// Update the object directly since status subresource is complex with fake client
	if err := fakeClient.Update(ctx, &createdGuild); err != nil {
		t.Fatalf("Failed to update guild with status: %v", err)
	}

	// Verify final state
	var finalGuild guildv1alpha1.Guild
	if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(guild), &finalGuild); err != nil {
		t.Fatalf("Failed to get final guild state: %v", err)
	}

	if meta.GetExternalName(&finalGuild) != "123456789" {
		t.Errorf("Expected external name '123456789', got '%s'", meta.GetExternalName(&finalGuild))
	}

	if finalGuild.Status.AtProvider.ID != "123456789" {
		t.Errorf("Expected status ID '123456789', got '%s'", finalGuild.Status.AtProvider.ID)
	}

	// Test deletion
	if err := fakeClient.Delete(ctx, &finalGuild); err != nil {
		t.Fatalf("Failed to delete guild: %v", err)
	}

	t.Log("Guild lifecycle test completed successfully")
}

func TestChannelLifecycle(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = apis.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	ctx := context.Background()

	// Test Channel creation
	channel := &channelv1alpha1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-channel",
			Namespace: "default",
		},
		Spec: channelv1alpha1.ChannelSpec{
			ForProvider: channelv1alpha1.ChannelParameters{
				Name:    "test-channel",
				Type:    0,
				GuildID: "123456789",
			},
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{
					Name: "test-provider-config",
				},
			},
		},
	}

	// Create the channel resource
	if err := fakeClient.Create(ctx, channel); err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Verify it was created
	var createdChannel channelv1alpha1.Channel
	if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(channel), &createdChannel); err != nil {
		t.Fatalf("Failed to get created channel: %v", err)
	}

	// Test setting external name (simulating controller behavior)
	meta.SetExternalName(&createdChannel, "987654321")
	if err := fakeClient.Update(ctx, &createdChannel); err != nil {
		t.Fatalf("Failed to update channel with external name: %v", err)
	}

	// Test status update (simulating controller behavior)
	// For integration tests, we'll just update the object directly
	createdChannel.Status.AtProvider = channelv1alpha1.ChannelObservation{
		ID:      "987654321",
		Name:    "test-channel",
		Type:    0,
		GuildID: "123456789",
	}
	createdChannel.Status.Conditions = []xpv1.Condition{
		{
			Type:               xpv1.TypeReady,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Reason:             "Available",
		},
	}
	// Update the object directly since status subresource is complex with fake client
	if err := fakeClient.Update(ctx, &createdChannel); err != nil {
		t.Fatalf("Failed to update channel with status: %v", err)
	}

	// Verify final state
	var finalChannel channelv1alpha1.Channel
	if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(channel), &finalChannel); err != nil {
		t.Fatalf("Failed to get final channel state: %v", err)
	}

	if meta.GetExternalName(&finalChannel) != "987654321" {
		t.Errorf("Expected external name '987654321', got '%s'", meta.GetExternalName(&finalChannel))
	}

	if finalChannel.Status.AtProvider.ID != "987654321" {
		t.Errorf("Expected status ID '987654321', got '%s'", finalChannel.Status.AtProvider.ID)
	}

	// Test deletion
	if err := fakeClient.Delete(ctx, &finalChannel); err != nil {
		t.Fatalf("Failed to delete channel: %v", err)
	}

	t.Log("Channel lifecycle test completed successfully")
}

func TestProviderConfigResourceLifecycle(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = apis.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	ctx := context.Background()

	// Test ProviderConfig creation
	providerConfig := &v1beta1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-provider-config",
		},
		Spec: v1beta1.ProviderConfigSpec{
			Credentials: v1beta1.ProviderCredentials{
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
		},
	}

	// Create the provider config
	if err := fakeClient.Create(ctx, providerConfig); err != nil {
		t.Fatalf("Failed to create provider config: %v", err)
	}

	// Verify it was created
	var createdProviderConfig v1beta1.ProviderConfig
	if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(providerConfig), &createdProviderConfig); err != nil {
		t.Fatalf("Failed to get created provider config: %v", err)
	}

	// Verify the spec
	if createdProviderConfig.Spec.Credentials.Source != xpv1.CredentialsSourceSecret {
		t.Errorf("Expected source to be secret, got %s", createdProviderConfig.Spec.Credentials.Source)
	}

	if createdProviderConfig.Spec.Credentials.SecretRef.Name != "discord-secret" {
		t.Errorf("Expected secret name 'discord-secret', got '%s'", createdProviderConfig.Spec.Credentials.SecretRef.Name)
	}

	t.Log("ProviderConfig lifecycle test completed successfully")
}

func TestIntegration(t *testing.T) {
	t.Run("GuildLifecycle", TestGuildLifecycle)
	t.Run("ChannelLifecycle", TestChannelLifecycle)
	t.Run("ProviderConfigResourceLifecycle", TestProviderConfigResourceLifecycle)
}
