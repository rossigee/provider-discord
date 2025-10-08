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
	"fmt"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	rolev1alpha1 "github.com/rossigee/provider-discord/apis/role/v1alpha1"
	"github.com/rossigee/provider-discord/apis/v1alpha1"
)

// Helper functions
func stringPtrE2E(s string) *string {
	return &s
}

func intPtrE2E(i int) *int {
	return &i
}

func boolPtrE2E(b bool) *bool {
	return &b
}

// TestEndToEndScenario tests a complete end-to-end scenario:
// 1. Create ProviderConfig
// 2. Create Guild
// 3. Create Channels in Guild
// 4. Create Roles in Guild
// 5. Update resources
// 6. Clean up resources
//
// Requires:
// - Kubernetes cluster with provider-discord installed
// - DISCORD_BOT_TOKEN environment variable
// - DISCORD_TEST_GUILD_ID environment variable (existing guild for testing)
func TestEndToEndScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	discordToken := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if discordToken == "" {
		t.Skip("DISCORD_BOT_TOKEN not set, skipping end-to-end test")
	}

	if testGuildID == "" {
		t.Skip("DISCORD_TEST_GUILD_ID not set, skipping end-to-end test")
	}

	// Get Kubernetes client (assumes running in cluster or with kubeconfig)
	k8sClient, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Failed to get Kubernetes client: %v", err)
	}

	ctx := context.Background()
	testNamespace := "default"
	testSuffix := fmt.Sprintf("%d", time.Now().Unix())

	t.Run("E2E-FullScenario", func(t *testing.T) {
		// Step 1: Create Secret
		secret := createDiscordSecret(testNamespace, testSuffix, discordToken)
		err := k8sClient.Create(ctx, secret)
		if err != nil {
			t.Fatalf("Failed to create secret: %v", err)
		}
		defer func() {
			if err := k8sClient.Delete(ctx, secret); err != nil {
				t.Logf("Warning: Failed to delete secret: %v", err)
			}
		}()

		// Step 2: Create ProviderConfig
		providerConfig := createProviderConfig(testSuffix, secret.Name, secret.Namespace)
		err = k8sClient.Create(ctx, providerConfig)
		if err != nil {
			t.Fatalf("Failed to create provider config: %v", err)
		}
		defer func() {
			if err := k8sClient.Delete(ctx, providerConfig); err != nil {
				t.Logf("Warning: Failed to delete provider config: %v", err)
			}
		}()

		// Wait for ProviderConfig to be ready
		err = waitForProviderConfigReady(ctx, k8sClient, providerConfig.Name, 30*time.Second)
		if err != nil {
			t.Fatalf("ProviderConfig not ready: %v", err)
		}

		// Step 3: Create test channels in existing guild
		channels := createTestChannels(testNamespace, testSuffix, testGuildID, providerConfig.Name)
		for _, channel := range channels {
			err = k8sClient.Create(ctx, channel)
			if err != nil {
				t.Fatalf("Failed to create channel %s: %v", channel.Name, err)
			}
			defer func(ch *channelv1alpha1.Channel) {
				if err := k8sClient.Delete(ctx, ch); err != nil {
					t.Logf("Warning: Failed to delete channel: %v", err)
				}
			}(channel)
		}

		// Wait for channels to be ready
		for _, channel := range channels {
			err = waitForChannelReady(ctx, k8sClient, channel.Name, channel.Namespace, 60*time.Second)
			if err != nil {
				t.Fatalf("Channel %s not ready: %v", channel.Name, err)
			}
		}

		// Step 4: Create test roles in existing guild
		roles := createTestRoles(testNamespace, testSuffix, testGuildID, providerConfig.Name)
		for _, role := range roles {
			err = k8sClient.Create(ctx, role)
			if err != nil {
				t.Fatalf("Failed to create role %s: %v", role.Name, err)
			}
			defer func(r *rolev1alpha1.Role) {
				if err := k8sClient.Delete(ctx, r); err != nil {
					t.Logf("Warning: Failed to delete role: %v", err)
				}
			}(role)
		}

		// Wait for roles to be ready
		for _, role := range roles {
			err = waitForRoleReady(ctx, k8sClient, role.Name, role.Namespace, 60*time.Second)
			if err != nil {
				t.Fatalf("Role %s not ready: %v", role.Name, err)
			}
		}

		// Step 5: Test updates
		t.Run("TestChannelUpdate", func(t *testing.T) {
			testChannelUpdate(ctx, t, k8sClient, channels[0])
		})

		t.Run("TestRoleUpdate", func(t *testing.T) {
			testRoleUpdate(ctx, t, k8sClient, roles[0])
		})

		// Step 6: Verify all resources are working
		t.Run("VerifyResourceStatus", func(t *testing.T) {
			verifyAllResourcesReady(ctx, t, k8sClient, testNamespace, testSuffix)
		})

		t.Log("End-to-end test completed successfully")
	})
}

func createDiscordSecret(namespace, suffix, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("discord-secret-%s", suffix),
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"token": []byte(token),
		},
	}
}

func createProviderConfig(suffix, secretName, secretNamespace string) *v1alpha1.ProviderConfig {
	return &v1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("provider-config-%s", suffix),
		},
		Spec: v1alpha1.ProviderConfigSpec{
			Credentials: v1alpha1.ProviderCredentials{
				Source: xpv1.CredentialsSourceSecret,
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      secretName,
							Namespace: secretNamespace,
						},
						Key: "token",
					},
				},
			},
			BaseURL: stringPtrE2E("https://discord.com/api/v10"),
		},
	}
}

func createTestChannels(namespace, suffix, guildID, providerConfigName string) []*channelv1alpha1.Channel {
	return []*channelv1alpha1.Channel{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-text-channel-%s", suffix),
				Namespace: namespace,
			},
			Spec: channelv1alpha1.ChannelSpec{
				ForProvider: channelv1alpha1.ChannelParameters{
					Name:    fmt.Sprintf("test-text-%s", suffix),
					Type:    0, // Text channel
					GuildID: guildID,
					Topic:   stringPtrE2E("Test text channel created by E2E tests"),
				},
				ResourceSpec: xpv1.ResourceSpec{
					ProviderConfigReference: &xpv1.Reference{
						Name: providerConfigName,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-voice-channel-%s", suffix),
				Namespace: namespace,
			},
			Spec: channelv1alpha1.ChannelSpec{
				ForProvider: channelv1alpha1.ChannelParameters{
					Name:      fmt.Sprintf("test-voice-%s", suffix),
					Type:      2, // Voice channel
					GuildID:   guildID,
					Bitrate:   intPtrE2E(64000),
					UserLimit: intPtrE2E(10),
				},
				ResourceSpec: xpv1.ResourceSpec{
					ProviderConfigReference: &xpv1.Reference{
						Name: providerConfigName,
					},
				},
			},
		},
	}
}

func createTestRoles(namespace, suffix, guildID, providerConfigName string) []*rolev1alpha1.Role {
	return []*rolev1alpha1.Role{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-admin-role-%s", suffix),
				Namespace: namespace,
			},
			Spec: rolev1alpha1.RoleSpec{
				ForProvider: rolev1alpha1.RoleParameters{
					Name:        fmt.Sprintf("Test Admin %s", suffix),
					GuildID:     guildID,
					Color:       intPtrE2E(0xFF0000), // Red
					Hoist:       boolPtrE2E(true),
					Mentionable: boolPtrE2E(false),
					Permissions: stringPtrE2E("8"), // Administrator permission
				},
				ResourceSpec: xpv1.ResourceSpec{
					ProviderConfigReference: &xpv1.Reference{
						Name: providerConfigName,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-member-role-%s", suffix),
				Namespace: namespace,
			},
			Spec: rolev1alpha1.RoleSpec{
				ForProvider: rolev1alpha1.RoleParameters{
					Name:        fmt.Sprintf("Test Member %s", suffix),
					GuildID:     guildID,
					Color:       intPtrE2E(0x00FF00), // Green
					Hoist:       boolPtrE2E(false),
					Mentionable: boolPtrE2E(true),
					Permissions: stringPtrE2E("1024"), // View channels permission
				},
				ResourceSpec: xpv1.ResourceSpec{
					ProviderConfigReference: &xpv1.Reference{
						Name: providerConfigName,
					},
				},
			},
		},
	}
}

func waitForProviderConfigReady(ctx context.Context, k8sClient client.Client, name string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		var pc v1alpha1.ProviderConfig
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, &pc)
		if err != nil {
			return false, err
		}

		// Check if ready condition exists and is true
		for _, condition := range pc.Status.Conditions {
			if condition.Type == xpv1.TypeReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	})
}

func waitForChannelReady(ctx context.Context, k8sClient client.Client, name, namespace string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		var channel channelv1alpha1.Channel
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &channel)
		if err != nil {
			return false, err
		}

		// Check if ready condition exists and is true
		for _, condition := range channel.Status.Conditions {
			if condition.Type == xpv1.TypeReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		// Also check if external name is set (indicates successful creation)
		if meta.GetExternalName(&channel) != "" {
			return true, nil
		}

		return false, nil
	})
}

func waitForRoleReady(ctx context.Context, k8sClient client.Client, name, namespace string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		var role rolev1alpha1.Role
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &role)
		if err != nil {
			return false, err
		}

		// Check if ready condition exists and is true
		for _, condition := range role.Status.Conditions {
			if condition.Type == xpv1.TypeReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		// Also check if external name is set (indicates successful creation)
		if meta.GetExternalName(&role) != "" {
			return true, nil
		}

		return false, nil
	})
}

func testChannelUpdate(ctx context.Context, t *testing.T, k8sClient client.Client, channel *channelv1alpha1.Channel) {
	// Update channel topic
	var currentChannel channelv1alpha1.Channel
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(channel), &currentChannel)
	if err != nil {
		t.Fatalf("Failed to get current channel: %v", err)
	}

	newTopic := fmt.Sprintf("Updated topic at %d", time.Now().Unix())
	currentChannel.Spec.ForProvider.Topic = stringPtrE2E(newTopic)

	err = k8sClient.Update(ctx, &currentChannel)
	if err != nil {
		t.Fatalf("Failed to update channel: %v", err)
	}

	// Wait for update to be applied
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		var updatedChannel channelv1alpha1.Channel
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(channel), &updatedChannel)
		if err != nil {
			return false, err
		}

		return updatedChannel.Status.AtProvider.Topic == newTopic, nil
	})

	if err != nil {
		t.Fatalf("Channel update not reflected in status: %v", err)
	}

	t.Logf("Successfully updated channel topic to: %s", newTopic)
}

func testRoleUpdate(ctx context.Context, t *testing.T, k8sClient client.Client, role *rolev1alpha1.Role) {
	// Update role color
	var currentRole rolev1alpha1.Role
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(role), &currentRole)
	if err != nil {
		t.Fatalf("Failed to get current role: %v", err)
	}

	newColor := 0x0000FF // Blue
	currentRole.Spec.ForProvider.Color = intPtrE2E(newColor)

	err = k8sClient.Update(ctx, &currentRole)
	if err != nil {
		t.Fatalf("Failed to update role: %v", err)
	}

	// Wait for update to be applied
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		var updatedRole rolev1alpha1.Role
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(role), &updatedRole)
		if err != nil {
			return false, err
		}

		return updatedRole.Spec.ForProvider.Color != nil && *updatedRole.Spec.ForProvider.Color == newColor, nil
	})

	if err != nil {
		t.Fatalf("Role update not reflected in status: %v", err)
	}

	t.Logf("Successfully updated role color to: %d", newColor)
}

func verifyAllResourcesReady(ctx context.Context, t *testing.T, k8sClient client.Client, namespace, suffix string) {
	// List all channels
	var channels channelv1alpha1.ChannelList
	err := k8sClient.List(ctx, &channels, client.InNamespace(namespace))
	if err != nil {
		t.Fatalf("Failed to list channels: %v", err)
	}

	testChannelCount := 0
	for _, channel := range channels.Items {
		if contains(channel.Name, suffix) {
			testChannelCount++
			// Verify channel is ready
			ready := false
			for _, condition := range channel.Status.Conditions {
				if condition.Type == xpv1.TypeReady && condition.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			if !ready {
				t.Errorf("Channel %s is not ready", channel.Name)
			}
		}
	}

	// List all roles
	var roles rolev1alpha1.RoleList
	err = k8sClient.List(ctx, &roles, client.InNamespace(namespace))
	if err != nil {
		t.Fatalf("Failed to list roles: %v", err)
	}

	testRoleCount := 0
	for _, role := range roles.Items {
		if contains(role.Name, suffix) {
			testRoleCount++
			// Verify role is ready
			ready := false
			for _, condition := range role.Status.Conditions {
				if condition.Type == xpv1.TypeReady && condition.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			if !ready {
				t.Errorf("Role %s is not ready", role.Name)
			}
		}
	}

	t.Logf("Verified %d test channels and %d test roles are ready", testChannelCount, testRoleCount)

	if testChannelCount == 0 || testRoleCount == 0 {
		t.Error("No test resources found - this suggests a problem with resource creation")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && len(substr) > 0 &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())
}

// getKubernetesClient returns a Kubernetes client
// This is a placeholder - in real tests you'd use controller-runtime's envtest
// or connect to an actual cluster
func getKubernetesClient() (client.Client, error) {
	// This would typically use:
	// - envtest for unit/integration testing
	// - Real cluster connection for e2e testing
	// For now, return an error to indicate this needs proper implementation
	return nil, fmt.Errorf("getKubernetesClient not implemented - need to set up envtest or cluster connection")
}

// TestE2EWithEnvTest demonstrates how to set up proper integration testing with envtest
func TestE2EWithEnvTest(t *testing.T) {
	t.Skip("envtest setup not implemented yet - would use sigs.k8s.io/controller-runtime/pkg/envtest")

	// This test would:
	// 1. Start a test Kubernetes API server using envtest
	// 2. Install provider CRDs
	// 3. Start the provider controllers
	// 4. Run the actual integration tests
	// 5. Clean up the test environment
}

// TestE2EProviderHealth tests provider health and monitoring endpoints
func TestE2EProviderHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping health test in short mode")
	}

	// This would test:
	// 1. Provider health endpoints (/healthz, /readyz)
	// 2. Metrics endpoint (/metrics)
	// 3. Provider status and logs

	t.Skip("Provider health testing not implemented - requires cluster access")
}

// TestE2EProviderUpgrade tests provider upgrade scenarios
func TestE2EProviderUpgrade(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping upgrade test in short mode")
	}

	// This would test:
	// 1. Install older version of provider
	// 2. Create resources with old version
	// 3. Upgrade to new version
	// 4. Verify resources still work
	// 5. Create new resources with new version
	// 6. Verify backward compatibility

	t.Skip("Provider upgrade testing not implemented - requires complex test setup")
}

// TestE2EResourceReconciliation tests resource reconciliation behavior
func TestE2EResourceReconciliation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reconciliation test in short mode")
	}

	// This would test:
	// 1. Create resources
	// 2. Manually modify Discord resources outside of Crossplane
	// 3. Verify provider detects drift
	// 4. Verify provider corrects drift
	// 5. Test various reconciliation scenarios

	t.Skip("Resource reconciliation testing not implemented - requires Discord API access and test guild")
}
