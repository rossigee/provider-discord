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

package webhook

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	webhookv1alpha1 "github.com/rossigee/provider-discord/apis/webhook/v1alpha1"
	"github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotWebhook   = "managed resource is not a Webhook custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

var (
	// Discord snowflake IDs are 18-19 digit numbers
	discordSnowflakeRegex = regexp.MustCompile(`^\d{18,19}$`)
)

// isValidDiscordID checks if the provided string is a valid Discord snowflake ID
func isValidDiscordID(id string) bool {
	return discordSnowflakeRegex.MatchString(id)
}

// Setup adds a controller that reconciles Webhook managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(webhookv1alpha1.WebhookGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(webhookv1alpha1.WebhookGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: clients.NewDiscordClient,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&webhookv1alpha1.Webhook{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	newServiceFn func(token string) *clients.DiscordClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*webhookv1alpha1.Webhook)
	if !ok {
		return nil, errors.New(errNotWebhook)
	}

	if cr.GetProviderConfigReference() == nil {
		return nil, errors.New("no providerConfigRef provided")
	}

	token, err := clients.GetConfig(ctx, c.kube, cr)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get discord config")
	}

	svc := c.newServiceFn(*token)

	return &external{service: svc, kube: c.kube}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	service clients.WebhookClient
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*webhookv1alpha1.Webhook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWebhook)
	}

	externalName := meta.GetExternalName(cr)

	// If external-name is empty or not a valid Discord ID, this is a new resource to be created
	// Crossplane runtime defaults external-name to metadata.name for new resources
	if externalName == "" || !isValidDiscordID(externalName) {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// If we have a valid external name (Discord webhook ID), try to get by ID
	webhook, err := c.service.GetWebhook(ctx, externalName)
	if err != nil {
		// If webhook not found by ID, assume it needs to be created
		// This handles cases where external-name was set but webhook doesn't exist
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	if webhook == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Update status with observed values
	now := &metav1.Time{Time: time.Now()}
	observation := webhookv1alpha1.WebhookObservation{
		ID:        webhook.ID,
		Type:      webhook.Type,
		Name:      webhook.Name,
		ChannelID: webhook.ChannelID,
		GuildID:   webhook.GuildID,
		UpdatedAt: now,
	}

	// Handle optional fields
	if webhook.Avatar != nil {
		observation.Avatar = *webhook.Avatar
	}
	if webhook.ApplicationID != nil {
		observation.ApplicationID = *webhook.ApplicationID
	}

	cr.Status.AtProvider = observation

	// Store sensitive fields in connection secret
	connectionDetails := managed.ConnectionDetails{}
	if webhook.Token != "" {
		connectionDetails["token"] = []byte(webhook.Token)
	}
	if webhook.URL != "" {
		connectionDetails["url"] = []byte(webhook.URL)
	}

	// Check if we need to update
	needsUpdate := cr.Spec.ForProvider.Name != webhook.Name ||
		(cr.Spec.ForProvider.Avatar != nil && (webhook.Avatar == nil || *cr.Spec.ForProvider.Avatar != *webhook.Avatar))

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  !needsUpdate,
		ConnectionDetails: connectionDetails,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*webhookv1alpha1.Webhook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWebhook)
	}

	cr.SetConditions(xpv1.Creating())

	req := &clients.CreateWebhookRequest{
		Name:   cr.Spec.ForProvider.Name,
		Avatar: cr.Spec.ForProvider.Avatar,
	}

	webhook, err := c.service.CreateWebhook(ctx, cr.Spec.ForProvider.ChannelID, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create webhook")
	}

	meta.SetExternalName(cr, webhook.ID)

	// Store sensitive fields in connection secret
	connectionDetails := managed.ConnectionDetails{}
	if webhook.Token != "" {
		connectionDetails["token"] = []byte(webhook.Token)
	}
	if webhook.URL != "" {
		connectionDetails["url"] = []byte(webhook.URL)
	}

	return managed.ExternalCreation{
		ConnectionDetails: connectionDetails,
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*webhookv1alpha1.Webhook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWebhook)
	}

	req := &clients.ModifyWebhookRequest{
		Name: &cr.Spec.ForProvider.Name,
	}

	// Set optional fields for update
	if cr.Spec.ForProvider.Avatar != nil {
		req.Avatar = cr.Spec.ForProvider.Avatar
	}

	_, err := c.service.ModifyWebhook(ctx, meta.GetExternalName(cr), req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update webhook")
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*webhookv1alpha1.Webhook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWebhook)
	}

	cr.SetConditions(xpv1.Deleting())

	err := c.service.DeleteWebhook(ctx, meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete webhook")
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// Nothing to disconnect for Discord API client
	return nil
}
