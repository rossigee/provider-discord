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

package garbagecollection

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	discordv1alpha1 "github.com/rossigee/provider-discord/apis/v1alpha1"
	"github.com/rossigee/provider-discord/internal/services"
)

const (
	controllerName = "garbagecollection"
	finalizerName  = "garbagecollection.discord.crossplane.io/finalizer"
)

// ProviderConfigReconciler reconciles ProviderConfig objects with garbage collection enabled.
type ProviderConfigReconciler struct {
	client   client.Client
	recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=discord.crossplane.io,resources=providerconfigs,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=discord.crossplane.io,resources=providerconfigs/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=discord.crossplane.io,resources=providerconfigusages,verbs=get;list
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// SetupWithManager sets up the controller with the Manager.
func (r *ProviderConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(controllerName)

	return ctrl.NewControllerManagedBy(mgr).
		For(&discordv1alpha1.ProviderConfig{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			pc := obj.(*discordv1alpha1.ProviderConfig)
			return pc.Spec.GarbageCollection != nil && pc.Spec.GarbageCollection.Enabled
		})).
		Complete(r)
}

// Reconcile performs periodic garbage collection on the ProviderConfig's Discord resources.
func (r *ProviderConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pc := &discordv1alpha1.ProviderConfig{}
	if err := r.client.Get(ctx, req.NamespacedName, pc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if pc.Spec.GarbageCollection == nil || !pc.Spec.GarbageCollection.Enabled {
		return ctrl.Result{}, nil
	}

	// Extract credentials
	botToken, baseURL, err := extractCredentials(ctx, r.client, pc)
	if err != nil {
		r.recorder.Eventf(pc, corev1.EventTypeWarning, "GCFailed", "Failed to extract credentials: %v", err)
		return ctrl.Result{}, err
	}

	// Run garbage collection
	gcService := services.NewGarbageCollectionService(botToken, baseURL, r.client)
	result, err := gcService.RunGarbageCollection(ctx, pc.Spec.GarbageCollection)
	if err != nil {
		r.recorder.Eventf(pc, corev1.EventTypeWarning, "GCFailed", "Garbage collection failed: %v", err)
		return ctrl.Result{}, err
	}

	// Record results
	eventMsg := fmt.Sprintf("Garbage collection completed: %d duplicates prevented, %d duplicates deleted, %d orphaned resources cleaned",
		result.DuplicatesPrevented, result.DuplicatesDeleted, result.OrphanedResourcesDeleted)

	if result.HasErrors {
		eventMsg += fmt.Sprintf(" (with %d errors)", len(result.Errors))
		r.recorder.Eventf(pc, corev1.EventTypeWarning, "GCCompleted", "%s", eventMsg)
	} else {
		r.recorder.Eventf(pc, corev1.EventTypeNormal, "GCCompleted", "%s", eventMsg)
	}

	// Requeue after configured interval (or default to 5 minutes)
	interval := time.Duration(300) * time.Second // Default 5 minutes
	if pc.Spec.GarbageCollection.PollIntervalSeconds != nil && *pc.Spec.GarbageCollection.PollIntervalSeconds > 0 {
		interval = time.Duration(*pc.Spec.GarbageCollection.PollIntervalSeconds) * time.Second
	}

	return ctrl.Result{RequeueAfter: interval}, nil
}

// extractCredentials extracts the bot token and base URL from the ProviderConfig.
func extractCredentials(ctx context.Context, c client.Client, pc *discordv1alpha1.ProviderConfig) (string, string, error) {
	secretRef := pc.Spec.Credentials.SecretRef
	if secretRef == nil {
		return "", "", fmt.Errorf("no credentials secret reference found")
	}

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}
	if err := c.Get(ctx, secretKey, secret); err != nil {
		return "", "", fmt.Errorf("failed to get credentials secret: %w", err)
	}

	key := secretRef.Key
	if key == "" {
		key = "token"
	}

	token := string(secret.Data[key])
	if token == "" {
		return "", "", fmt.Errorf("empty token in secret %s/%s key %s", secretRef.Namespace, secretRef.Name, key)
	}

	baseURL := "https://discord.com/api/v10"
	if pc.Spec.BaseURL != nil && *pc.Spec.BaseURL != "" {
		baseURL = *pc.Spec.BaseURL
	}

	return token, baseURL, nil
}
