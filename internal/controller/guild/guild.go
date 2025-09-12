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

package guild

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

	guildv1alpha1 "github.com/rossigee/provider-discord/apis/guild/v1alpha1"
	"github.com/rossigee/provider-discord/apis/v1beta1"
	"github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotGuild     = "managed resource is not a Guild custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

// Setup adds a controller that reconciles Guild managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(guildv1alpha1.GuildGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(guildv1alpha1.GuildGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
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
		For(&guildv1alpha1.Guild{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(token string) *clients.DiscordClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*guildv1alpha1.Guild)
	if !ok {
		return nil, errors.New(errNotGuild)
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
	service clients.GuildClient
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*guildv1alpha1.Guild)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGuild)
	}

	// If we have an external name (guild ID), try to get by ID
	if meta.GetExternalName(cr) != "" {
		guild, err := c.service.GetGuild(ctx, meta.GetExternalName(cr))
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, "failed to get guild by ID")
		}

		if guild == nil {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}

		// Update status with observed values
		now := &metav1.Time{Time: time.Now()}
		cr.Status.AtProvider = guildv1alpha1.GuildObservation{
			ID:                          guild.ID,
			Name:                        guild.Name,
			OwnerID:                     guild.OwnerID,
			VerificationLevel:           guild.VerificationLevel,
			DefaultMessageNotifications: guild.DefaultMessageNotifications,
			ExplicitContentFilter:       guild.ExplicitContentFilter,
			Features:                    guild.Features,
			AFKTimeout:                  guild.AFKTimeout,
			SystemChannelFlags:          guild.SystemChannelFlags,
			UpdatedAt:                   now,
		}

		if guild.Region != nil {
			cr.Status.AtProvider.Region = *guild.Region
		}
		if guild.Icon != nil {
			cr.Status.AtProvider.Icon = *guild.Icon
		}
		if guild.AFKChannelID != nil {
			cr.Status.AtProvider.AFKChannelID = *guild.AFKChannelID
		}
		if guild.SystemChannelID != nil {
			cr.Status.AtProvider.SystemChannelID = *guild.SystemChannelID
		}
		if guild.ApproximateMemberCount != nil {
			cr.Status.AtProvider.MemberCount = *guild.ApproximateMemberCount
		}

		cr.SetConditions(xpv1.Available())

		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: c.isUpToDate(cr, guild),
			ConnectionDetails: managed.ConnectionDetails{
				"guildId":   []byte(guild.ID),
				"guildName": []byte(guild.Name),
			},
		}, nil
	}

	// No external name means the guild doesn't exist yet
	return managed.ExternalObservation{
		ResourceExists: false,
	}, nil
}

func (c *external) isUpToDate(cr *guildv1alpha1.Guild, guild *clients.Guild) bool {
	// Check if name needs to be updated
	if cr.Spec.ForProvider.Name != guild.Name {
		return false
	}

	// Check if region needs to be updated
	if cr.Spec.ForProvider.Region != nil {
		if guild.Region == nil || *cr.Spec.ForProvider.Region != *guild.Region {
			return false
		}
	}

	// Check if verification level needs to be updated
	if cr.Spec.ForProvider.VerificationLevel != nil {
		if *cr.Spec.ForProvider.VerificationLevel != guild.VerificationLevel {
			return false
		}
	}

	// Check if default message notifications needs to be updated
	if cr.Spec.ForProvider.DefaultMessageNotifications != nil {
		if *cr.Spec.ForProvider.DefaultMessageNotifications != guild.DefaultMessageNotifications {
			return false
		}
	}

	// Check if explicit content filter needs to be updated
	if cr.Spec.ForProvider.ExplicitContentFilter != nil {
		if *cr.Spec.ForProvider.ExplicitContentFilter != guild.ExplicitContentFilter {
			return false
		}
	}

	// Check if AFK timeout needs to be updated
	if cr.Spec.ForProvider.AFKTimeout != nil {
		if *cr.Spec.ForProvider.AFKTimeout != guild.AFKTimeout {
			return false
		}
	}

	// Check if system channel flags needs to be updated
	if cr.Spec.ForProvider.SystemChannelFlags != nil {
		if *cr.Spec.ForProvider.SystemChannelFlags != guild.SystemChannelFlags {
			return false
		}
	}

	return true
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*guildv1alpha1.Guild)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGuild)
	}

	cr.SetConditions(xpv1.Creating())

	req := &clients.CreateGuildRequest{
		Name: cr.Spec.ForProvider.Name,
	}

	if cr.Spec.ForProvider.Region != nil {
		req.Region = cr.Spec.ForProvider.Region
	}
	if cr.Spec.ForProvider.Icon != nil {
		req.Icon = cr.Spec.ForProvider.Icon
	}
	if cr.Spec.ForProvider.VerificationLevel != nil {
		req.VerificationLevel = cr.Spec.ForProvider.VerificationLevel
	}
	if cr.Spec.ForProvider.DefaultMessageNotifications != nil {
		req.DefaultMessageNotifications = cr.Spec.ForProvider.DefaultMessageNotifications
	}
	if cr.Spec.ForProvider.ExplicitContentFilter != nil {
		req.ExplicitContentFilter = cr.Spec.ForProvider.ExplicitContentFilter
	}
	if cr.Spec.ForProvider.AFKChannelID != nil {
		req.AFKChannelID = cr.Spec.ForProvider.AFKChannelID
	}
	if cr.Spec.ForProvider.AFKTimeout != nil {
		req.AFKTimeout = cr.Spec.ForProvider.AFKTimeout
	}
	if cr.Spec.ForProvider.SystemChannelID != nil {
		req.SystemChannelID = cr.Spec.ForProvider.SystemChannelID
	}
	if cr.Spec.ForProvider.SystemChannelFlags != nil {
		req.SystemChannelFlags = cr.Spec.ForProvider.SystemChannelFlags
	}

	guild, err := c.service.CreateGuild(ctx, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create guild")
	}

	meta.SetExternalName(cr, guild.ID)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"guildId":   []byte(guild.ID),
			"guildName": []byte(guild.Name),
		},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*guildv1alpha1.Guild)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGuild)
	}

	req := &clients.ModifyGuildRequest{}
	needsUpdate := false

	// Check what fields need updating
	if cr.Spec.ForProvider.Name != cr.Status.AtProvider.Name {
		req.Name = &cr.Spec.ForProvider.Name
		needsUpdate = true
	}

	if cr.Spec.ForProvider.Region != nil && (cr.Status.AtProvider.Region == "" || *cr.Spec.ForProvider.Region != cr.Status.AtProvider.Region) {
		req.Region = cr.Spec.ForProvider.Region
		needsUpdate = true
	}

	if cr.Spec.ForProvider.VerificationLevel != nil && *cr.Spec.ForProvider.VerificationLevel != cr.Status.AtProvider.VerificationLevel {
		req.VerificationLevel = cr.Spec.ForProvider.VerificationLevel
		needsUpdate = true
	}

	if cr.Spec.ForProvider.DefaultMessageNotifications != nil && *cr.Spec.ForProvider.DefaultMessageNotifications != cr.Status.AtProvider.DefaultMessageNotifications {
		req.DefaultMessageNotifications = cr.Spec.ForProvider.DefaultMessageNotifications
		needsUpdate = true
	}

	if cr.Spec.ForProvider.ExplicitContentFilter != nil && *cr.Spec.ForProvider.ExplicitContentFilter != cr.Status.AtProvider.ExplicitContentFilter {
		req.ExplicitContentFilter = cr.Spec.ForProvider.ExplicitContentFilter
		needsUpdate = true
	}

	if cr.Spec.ForProvider.AFKTimeout != nil && *cr.Spec.ForProvider.AFKTimeout != cr.Status.AtProvider.AFKTimeout {
		req.AFKTimeout = cr.Spec.ForProvider.AFKTimeout
		needsUpdate = true
	}

	if cr.Spec.ForProvider.SystemChannelFlags != nil && *cr.Spec.ForProvider.SystemChannelFlags != cr.Status.AtProvider.SystemChannelFlags {
		req.SystemChannelFlags = cr.Spec.ForProvider.SystemChannelFlags
		needsUpdate = true
	}

	if needsUpdate {
		_, err := c.service.ModifyGuild(ctx, meta.GetExternalName(cr), req)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update guild")
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*guildv1alpha1.Guild)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotGuild)
	}

	cr.SetConditions(xpv1.Deleting())

	err := c.service.DeleteGuild(ctx, meta.GetExternalName(cr))
	if err != nil {
		// Check if the error is a 404 (guild not found), which means it's already deleted
		// This is a simplified error check - in production, you'd want more robust error handling
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete guild")
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// Nothing to disconnect for Discord API client
	return nil
}