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

package channel

import (
	"context"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	"github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotChannel = "managed resource is not a Channel custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

// Setup adds a controller that reconciles Channel managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(channelv1alpha1.ChannelGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(channelv1alpha1.ChannelGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: clients.NewDiscordClient,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&channelv1alpha1.Channel{}).
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
	cr, ok := mg.(*channelv1alpha1.Channel)
	if !ok {
		return nil, errors.New(errNotChannel)
	}

	if mg.GetProviderConfigReference() == nil {
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
	service clients.ChannelClient
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*channelv1alpha1.Channel)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotChannel)
	}

	// If we have an external name (channel ID), try to get by ID
	if meta.GetExternalName(cr) != "" {
		channel, err := c.service.GetChannel(ctx, meta.GetExternalName(cr))
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, "failed to get channel by ID")
		}

		if channel == nil {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}

		// Update status with observed values
		now := &metav1.Time{Time: time.Now()}
		cr.Status.AtProvider = channelv1alpha1.ChannelObservation{
			ID:       channel.ID,
			Name:     channel.Name,
			Type:     channel.Type,
			GuildID:  channel.GuildID,
			Position: channel.Position,
			ParentID: channel.ParentID,
			UpdatedAt: now,
		}

		// Check if we need to update
		needsUpdate := false
		if cr.Spec.ForProvider.Name != channel.Name {
			needsUpdate = true
		}
		if cr.Spec.ForProvider.Position != nil && *cr.Spec.ForProvider.Position != channel.Position {
			needsUpdate = true
		}
		if cr.Spec.ForProvider.ParentID != nil && *cr.Spec.ForProvider.ParentID != channel.ParentID {
			needsUpdate = true
		}

		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: !needsUpdate,
		}, nil
	}

	// No external name means the resource doesn't exist
	return managed.ExternalObservation{
		ResourceExists: false,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*channelv1alpha1.Channel)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotChannel)
	}

	cr.SetConditions(xpv1.Creating())

	req := &clients.CreateChannelRequest{
		Name:     cr.Spec.ForProvider.Name,
		Type:     cr.Spec.ForProvider.Type,
		GuildID:  cr.Spec.ForProvider.GuildID,
		Position: cr.Spec.ForProvider.Position,
		ParentID: cr.Spec.ForProvider.ParentID,
	}

	// Set optional fields
	if cr.Spec.ForProvider.Topic != nil {
		req.Topic = cr.Spec.ForProvider.Topic
	}
	if cr.Spec.ForProvider.Bitrate != nil {
		req.Bitrate = cr.Spec.ForProvider.Bitrate
	}
	if cr.Spec.ForProvider.UserLimit != nil {
		req.UserLimit = cr.Spec.ForProvider.UserLimit
	}
	if cr.Spec.ForProvider.RateLimitPerUser != nil {
		req.RateLimitPerUser = cr.Spec.ForProvider.RateLimitPerUser
	}
	if cr.Spec.ForProvider.NSFW != nil {
		req.NSFW = cr.Spec.ForProvider.NSFW
	}

	channel, err := c.service.CreateChannel(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create channel")
	}

	meta.SetExternalName(cr, channel.ID)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*channelv1alpha1.Channel)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotChannel)
	}

	req := &clients.ModifyChannelRequest{
		Name: &cr.Spec.ForProvider.Name,
	}

	// Set optional fields for update
	if cr.Spec.ForProvider.Position != nil {
		req.Position = cr.Spec.ForProvider.Position
	}
	if cr.Spec.ForProvider.Topic != nil {
		req.Topic = cr.Spec.ForProvider.Topic
	}
	if cr.Spec.ForProvider.NSFW != nil {
		req.NSFW = cr.Spec.ForProvider.NSFW
	}
	if cr.Spec.ForProvider.ParentID != nil {
		req.ParentID = cr.Spec.ForProvider.ParentID
	}
	if cr.Spec.ForProvider.Bitrate != nil {
		req.Bitrate = cr.Spec.ForProvider.Bitrate
	}
	if cr.Spec.ForProvider.UserLimit != nil {
		req.UserLimit = cr.Spec.ForProvider.UserLimit
	}
	if cr.Spec.ForProvider.RateLimitPerUser != nil {
		req.RateLimitPerUser = cr.Spec.ForProvider.RateLimitPerUser
	}

	_, err := c.service.ModifyChannel(ctx, meta.GetExternalName(cr), req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update channel")
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*channelv1alpha1.Channel)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotChannel)
	}

	cr.SetConditions(xpv1.Deleting())

	err := c.service.DeleteChannel(ctx, meta.GetExternalName(cr))
	if err != nil {
		// Check if the error is a 404 (channel not found), which means it's already deleted
		// This is a simplified error check - in production, you'd want more robust error handling
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete channel")
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// Nothing to disconnect for Discord API client
	return nil
}