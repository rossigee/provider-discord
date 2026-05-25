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
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	deduplicationv1alpha1 "github.com/rossigee/provider-discord/apis/deduplication/v1alpha1"
	discordv1alpha1 "github.com/rossigee/provider-discord/apis/v1alpha1"
	"github.com/rossigee/provider-discord/internal/services"
)

const (
	// DeduplicationAnnotation is the annotation key for triggering deduplication.
	DeduplicationAnnotation = "discord.crossplane.io/deduplication"

	// DeduplicationFinalizerName is the finalizer used for cleanup.
	DeduplicationFinalizerName = "deduplication.discord.crossplane.io/cleanup"

	// lastProcessedAnnotation tracks the last processed mode to avoid duplicate work.
	lastProcessedAnnotation = "discord.crossplane.io/deduplication-processed"
)

// ProviderConfigReconciler reconciles ProviderConfig objects and handles deduplication.
type ProviderConfigReconciler struct {
	client.Client
	Recorder record.EventRecorder
}

// Setup adds the reconciler to the manager.
func Setup(mgr ctrl.Manager) error {
	r := &ProviderConfigReconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorderFor("discord-provider-deduplication"), //nolint:staticcheck
	}

	// Predicate to only watch ProviderConfigs with the deduplication annotation
	annotationPredicate := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		pc, ok := obj.(*discordv1alpha1.ProviderConfig)
		if !ok {
			return false
		}
		if pc.Annotations == nil {
			return false
		}
		mode, hasMode := pc.Annotations[DeduplicationAnnotation]
		if !hasMode {
			return false
		}
		// Only process if mode is "report" or "action"
		return mode == "report" || mode == "action"
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&discordv1alpha1.ProviderConfig{}).
		WithEventFilter(annotationPredicate).
		Complete(r)
}

// Reconcile handles reconciliation of ProviderConfig objects with deduplication annotations.
//
// Ordering guarantees (crash-safety):
//  1. Credentials are extracted first — purely read-only, safe to retry.
//  2. Deduplication CRD is created with phase="analyzing" BEFORE any Discord mutations,
//     establishing an audit trail even if later steps fail.
//  3. Idempotency annotation is patched onto ProviderConfig BEFORE the destructive
//     AnalyzeAndDeduplicate call. This prevents double-execution if the controller
//     restarts mid-operation (prefer at-most-once for irreversible channel deletions).
//  4. AnalyzeAndDeduplicate runs; CRD is updated with results afterwards.
func (r *ProviderConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Get the ProviderConfig
	pc := &discordv1alpha1.ProviderConfig{}
	if err := r.Get(ctx, req.NamespacedName, pc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if deduplication annotation is present and valid
	annotations := pc.GetAnnotations()
	if annotations == nil {
		return ctrl.Result{}, nil
	}

	mode, hasMode := annotations[DeduplicationAnnotation]
	if !hasMode || (mode != "report" && mode != "action") {
		return ctrl.Result{}, nil
	}

	// Check idempotency — skip if this exact mode was already processed
	lastProcessed := annotations[lastProcessedAnnotation]
	if lastProcessed == mode {
		log.V(4).Info("Deduplication already processed for this mode", "mode", mode)
		return ctrl.Result{}, nil
	}

	log.Info("Starting deduplication", "mode", mode)

	// Step 1: Extract credentials (read-only, safe to retry on failure)
	botToken, baseURL, err := r.extractCredentials(ctx, pc)
	if err != nil {
		log.Error(err, "failed to extract credentials")
		r.Recorder.Event(pc, corev1.EventTypeWarning, "DeduplicationFailed", fmt.Sprintf("Failed to extract credentials: %v", err))
		return ctrl.Result{}, err
	}

	// Resolve deduplication spec, defaulting if not provided
	spec := pc.Spec.Deduplication
	if spec == nil {
		spec = &discordv1alpha1.DeduplicationSpec{
			Enabled:                 true,
			Mode:                    discordv1alpha1.DeduplicationMode(mode),
			DeleteOrphanedResources: true,
			TargetGuilds:            []string{},
		}
	}
	if spec.TargetGuilds == nil {
		spec.TargetGuilds = []string{}
	}

	// Step 2: Create Deduplication CRD with phase="analyzing" BEFORE any Discord mutations.
	// The name is deterministic ({pc-name}-{mode}) so CreateOrUpdate correctly updates
	// the same object on every reconcile instead of leaking a new one each second.
	dedupName := fmt.Sprintf("%s-%s", req.Name, mode)
	dedupCRD := &deduplicationv1alpha1.Deduplication{
		ObjectMeta: metav1.ObjectMeta{Name: dedupName},
	}
	startTime := time.Now()
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, dedupCRD, func() error {
		dedupCRD.Spec.ProviderConfigRef = deduplicationv1alpha1.ProviderConfigReference{Name: pc.Name}
		dedupCRD.Spec.Mode = mode
		dedupCRD.Spec.DeleteOrphanedResources = spec.DeleteOrphanedResources
		dedupCRD.Spec.TargetGuilds = spec.TargetGuilds
		dedupCRD.Status.Phase = "analyzing"
		if dedupCRD.Status.StartTime == nil {
			dedupCRD.Status.StartTime = &metav1.Time{Time: startTime}
		}
		return nil
	})
	if err != nil {
		log.Error(err, "failed to create Deduplication CRD")
		return ctrl.Result{}, err
	}

	// Step 3: Stamp the idempotency annotation BEFORE the destructive operation so that
	// a mid-run restart does not trigger a second deletion pass.
	// Trade-off: if the operation is interrupted, a manual annotation reset is needed to retry.
	patchData := client.MergeFrom(pc.DeepCopy())
	if pc.Annotations == nil {
		pc.Annotations = make(map[string]string)
	}
	pc.Annotations[lastProcessedAnnotation] = mode
	if err := r.Patch(ctx, pc, patchData); err != nil {
		log.Error(err, "failed to set idempotency annotation")
		return ctrl.Result{}, err
	}

	// Step 4: Run the deduplication (potentially destructive in "action" mode)
	httpClient := &http.Client{Timeout: 30 * time.Second}
	dedupService := services.NewDeduplicationService(httpClient, baseURL, botToken, r.Client)

	result, err := dedupService.AnalyzeAndDeduplicate(ctx, mode, spec.TargetGuilds)
	if err != nil {
		log.Error(err, "deduplication analysis failed")
		r.Recorder.Event(pc, corev1.EventTypeWarning, "DeduplicationFailed", fmt.Sprintf("Analysis failed: %v", err))
		// Best-effort: update CRD to reflect failure so operators can see what happened
		_, _ = controllerutil.CreateOrUpdate(ctx, r.Client, dedupCRD, func() error {
			dedupCRD.Status.Phase = "failed"
			dedupCRD.Status.CompletionTime = &metav1.Time{Time: time.Now()}
			dedupCRD.Status.LastError = err.Error()
			return nil
		})
		return ctrl.Result{}, err
	}

	// Step 5: Update CRD with final results
	_, updateErr := controllerutil.CreateOrUpdate(ctx, r.Client, dedupCRD, func() error {
		dedupCRD.Status.Phase = "completed"
		dedupCRD.Status.CompletionTime = &metav1.Time{Time: time.Now()}
		dedupCRD.Status.Summary = result.Summary
		dedupCRD.Status.Results = make(map[string]deduplicationv1alpha1.GuildDeduplicationResult)

		for guildID, guildResult := range result.Guilds {
			dupGroups := make([]deduplicationv1alpha1.DuplicateGroupInfo, 0, len(guildResult.DuplicateGroups))
			for _, group := range guildResult.DuplicateGroups {
				dupInfo := deduplicationv1alpha1.DuplicateGroupInfo{
					ChannelName:   group.Name,
					Count:         len(group.Channels),
					KeptChannelID: group.Channels[group.KeepIndex].ID,
				}
				deletedIDs := make([]string, 0, len(group.Channels)-1)
				for i, ch := range group.Channels {
					if i != group.KeepIndex {
						deletedIDs = append(deletedIDs, ch.ID)
					}
				}
				dupInfo.DeletedChannelIDs = deletedIDs
				dupGroups = append(dupGroups, dupInfo)
			}
			dedupCRD.Status.Results[guildID] = deduplicationv1alpha1.GuildDeduplicationResult{
				GuildID:                  guildResult.GuildID,
				GuildName:                guildResult.GuildName,
				TotalChannels:            guildResult.TotalChannels,
				DuplicateGroups:          dupGroups,
				ChannelsDeleted:          guildResult.ChannelsDeleted,
				OrphanedResourcesDeleted: guildResult.OrphanedResourcesDeleted,
				Errors:                   guildResult.Errors,
			}
		}
		return nil
	})
	if updateErr != nil {
		// Non-fatal: the operation completed successfully; idempotency annotation is already set.
		log.Error(updateErr, "failed to update Deduplication CRD with results (operation still succeeded)")
	}

	// Record summary event
	eventMsg := fmt.Sprintf("Deduplication %q completed (mode=%s): %d guilds, %d duplicates found, %d channels deleted",
		dedupName, mode,
		result.Summary.TotalGuildsAnalyzed,
		result.Summary.TotalDuplicateChannelsFound,
		result.Summary.ChannelsDeleted)
	eventType := corev1.EventTypeNormal
	if result.HasError {
		eventType = corev1.EventTypeWarning
		eventMsg += fmt.Sprintf(". Error: %s", result.Error)
	}
	r.Recorder.Event(pc, eventType, "DeduplicationCompleted", eventMsg)

	log.Info("Deduplication completed", "mode", mode, "guilds", result.Summary.TotalGuildsAnalyzed,
		"duplicatesFound", result.Summary.TotalDuplicateChannelsFound,
		"channelsDeleted", result.Summary.ChannelsDeleted)

	return ctrl.Result{}, nil
}

// extractCredentials extracts the bot token and base URL from the ProviderConfig.
// It uses the configured SecretRef.Key first, then falls back to the common key names
// "token" and "credentials" for backward compatibility. The returned token is always
// whitespace-trimmed to handle secrets provisioned with trailing newlines.
func (r *ProviderConfigReconciler) extractCredentials(ctx context.Context, pc *discordv1alpha1.ProviderConfig) (string, string, error) {
	secretRef := pc.Spec.Credentials.SecretRef
	if secretRef == nil {
		return "", "", fmt.Errorf("no credentials secret reference found")
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}, secret); err != nil {
		return "", "", fmt.Errorf("failed to get credentials secret: %w", err)
	}

	// Resolve the token: use the explicitly configured key first (consistent with GetConfig),
	// then fall back to common key names for backward compatibility.
	var rawToken []byte
	var found bool
	candidateKeys := []string{secretRef.Key, "token", "credentials"}
	for _, key := range candidateKeys {
		if key == "" {
			continue
		}
		if rawToken, found = secret.Data[key]; found {
			break
		}
	}
	if !found {
		return "", "", fmt.Errorf("bot token not found in secret %s/%s (checked keys: %v)",
			secretRef.Namespace, secretRef.Name, candidateKeys)
	}

	// Trim whitespace/newlines that appear when secrets are provisioned with `echo TOKEN | base64`
	botToken := strings.TrimSpace(string(rawToken))
	if botToken == "" {
		return "", "", fmt.Errorf("bot token is empty in secret %s/%s", secretRef.Namespace, secretRef.Name)
	}

	baseURL := "https://discord.com/api/v10"
	if pc.Spec.BaseURL != nil {
		baseURL = *pc.Spec.BaseURL
	}

	return botToken, baseURL, nil
}
