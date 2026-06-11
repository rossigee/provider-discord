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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	"github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotChannel   = "managed resource is not a Channel custom resource"
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

// isDiscordNotFound reports whether a Discord API error is a 404 not-found response.
func isDiscordNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Discord API error: 404")
}

// checkChannelExistsByName checks if a channel with the same name already exists in the guild
func (c *external) checkChannelExistsByName(ctx context.Context, cr *channelv1alpha1.Channel) (managed.ExternalObservation, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(4).Info("Checking for existing channel by name", "name", cr.Spec.ForProvider.Name, "guildID", cr.Spec.ForProvider.GuildID)

	// List all channels in the guild
	channels, err := c.service.ListGuildChannels(ctx, cr.Spec.ForProvider.GuildID)
	if err != nil {
		// Return error instead of assuming non-existence to prevent duplicate creation
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to list guild channels")
	}

	// Check if any existing channel has the same name
	for _, channel := range channels {
		if channel.Name == cr.Spec.ForProvider.Name {
			log.V(4).Info("Found existing channel by name, adopting", "name", channel.Name, "id", channel.ID)

			// Set the external name to the existing channel's ID
			meta.SetExternalName(cr, channel.ID)

			// Update status with observed values
			now := &metav1.Time{Time: time.Now()}
			cr.Status.AtProvider = channelv1alpha1.ChannelObservation{
				ID:        channel.ID,
				Name:      channel.Name,
				Type:      channel.Type,
				GuildID:   channel.GuildID,
				Position:  channel.Position,
				ParentID:  channel.ParentID,
				UpdatedAt: now,
			}

			// Since we matched by name, only position and parentID can differ
			needsUpdate := false
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
	}

	log.V(4).Info("Channel not found by name, will create", "name", cr.Spec.ForProvider.Name)

	// Channel doesn't exist, needs to be created
	return managed.ExternalObservation{
		ResourceExists: false,
	}, nil
}

// Setup adds a controller that reconciles Channel managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return SetupWithClient(mgr, o, clients.NewDiscordClient)
}

// SetupWithClient adds a controller that reconciles Channel managed resources with a custom client factory.
func SetupWithClient(mgr ctrl.Manager, o controller.Options, newServiceFn func(token string) *clients.DiscordClient) error {
	name := managed.ControllerName(channelv1alpha1.ChannelGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(channelv1alpha1.ChannelGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: newServiceFn,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(name))))

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
	service clients.ChannelClient
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*channelv1alpha1.Channel)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotChannel)
	}

	log := ctrl.LoggerFrom(ctx)
	externalName := meta.GetExternalName(cr)

	log.V(4).Info("Observing channel", "externalName", externalName, "channelName", cr.Spec.ForProvider.Name)

	// If external-name is empty or not a valid Discord ID, check if channel exists by name.
	// Crossplane runtime defaults external-name to metadata.name for new resources.
	if externalName == "" {
		return c.checkChannelExistsByName(ctx, cr)
	}

	// Check if external-name is a valid Discord snowflake ID (18-19 digits)
	if !isValidDiscordID(externalName) {
		// For non-snowflake external names, check if channel exists by name
		return c.checkChannelExistsByName(ctx, cr)
	}

	// If we have a valid external name (Discord channel ID), try to get by ID
	channel, err := c.service.GetChannel(ctx, externalName)
	if err != nil {
		if isDiscordNotFound(err) {
			// Channel was deleted externally; let Crossplane recreate it
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		// Propagate transient errors (rate-limit, 5xx, network) so Crossplane
		// retries rather than accidentally provisioning a duplicate channel
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get channel")
	}

	if channel == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Update status with observed values
	now := &metav1.Time{Time: time.Now()}
	cr.Status.AtProvider = channelv1alpha1.ChannelObservation{
		ID:        channel.ID,
		Name:      channel.Name,
		Type:      channel.Type,
		GuildID:   channel.GuildID,
		Position:  channel.Position,
		ParentID:  channel.ParentID,
		UpdatedAt: now,
	}
	// Populate permission overwrites in status
	if len(channel.PermissionOverwrites) > 0 {
		cr.Status.AtProvider.PermissionOverwrites = make([]channelv1alpha1.PermissionOverwrite, len(channel.PermissionOverwrites))
		for i, pw := range channel.PermissionOverwrites {
			typeStr := "member"
			if pw.Type == 0 {
				typeStr = "role"
			}
			cr.Status.AtProvider.PermissionOverwrites[i] = channelv1alpha1.PermissionOverwrite{
				ID:   pw.ID,
				Type: typeStr,
			}
			if pw.Allow != "" {
				val, err := strconv.ParseInt(pw.Allow, 10, 64)
				if err == nil {
					cr.Status.AtProvider.PermissionOverwrites[i].Allow = &val
				}
			}
			if pw.Deny != "" {
				val, err := strconv.ParseInt(pw.Deny, 10, 64)
				if err == nil {
					cr.Status.AtProvider.PermissionOverwrites[i].Deny = &val
				}
			}
		}
	}

	// Late initialization: populate spec fields from observed state if not set
	lateInitialized := false
	if cr.Spec.ForProvider.GuildID == "" && channel.GuildID != "" {
		cr.Spec.ForProvider.GuildID = channel.GuildID
		lateInitialized = true
	}
	if cr.Spec.ForProvider.Type == 0 && channel.Type != 0 {
		cr.Spec.ForProvider.Type = channel.Type
		lateInitialized = true
	}

	// Check if we need to update
	needsUpdate := cr.Spec.ForProvider.Name != channel.Name
	if cr.Spec.ForProvider.Position != nil && *cr.Spec.ForProvider.Position != channel.Position {
		needsUpdate = true
	}
	if cr.Spec.ForProvider.ParentID != nil && *cr.Spec.ForProvider.ParentID != channel.ParentID {
		needsUpdate = true
	}
	// Check if permission overwrites differ
	if len(cr.Spec.ForProvider.PermissionOverwrites) != len(channel.PermissionOverwrites) {
		needsUpdate = true
	} else {
		for i, pw := range cr.Spec.ForProvider.PermissionOverwrites {
			if i >= len(channel.PermissionOverwrites) {
				needsUpdate = true
				break
			}
			channelPw := channel.PermissionOverwrites[i]
			if pw.ID != channelPw.ID {
				needsUpdate = true
				break
			}
			// Convert string allow/deny to int64 for comparison
			var channelAllow, channelDeny int64
			if channelPw.Allow != "" {
				if val, err := strconv.ParseInt(channelPw.Allow, 10, 64); err == nil {
					channelAllow = val
				}
			}
			if channelPw.Deny != "" {
				if val, err := strconv.ParseInt(channelPw.Deny, 10, 64); err == nil {
					channelDeny = val
				}
			}
			if pw.Allow != nil && channelPw.Allow == "" ||
				pw.Allow == nil && channelPw.Allow != "" ||
				pw.Allow != nil && channelPw.Allow != "" && *pw.Allow != channelAllow ||
				pw.Deny != nil && channelPw.Deny == "" ||
				pw.Deny == nil && channelPw.Deny != "" ||
				pw.Deny != nil && channelPw.Deny != "" && *pw.Deny != channelDeny {
				needsUpdate = true
				break
			}
		}
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        !needsUpdate,
		ResourceLateInitialized: lateInitialized,
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

	cr.SetConditions(xpv1.Available())

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
	if len(cr.Spec.ForProvider.PermissionOverwrites) > 0 {
		req.PermissionOverwrites = make([]clients.PermissionOverwrite, len(cr.Spec.ForProvider.PermissionOverwrites))
		for i, pw := range cr.Spec.ForProvider.PermissionOverwrites {
			var pType int
			if pw.Type == "role" {
				pType = 0
			} else {
				pType = 1
			}
			req.PermissionOverwrites[i] = clients.PermissionOverwrite{
				ID:   pw.ID,
				Type: pType,
			}
			if pw.Allow != nil {
				req.PermissionOverwrites[i].Allow = strconv.FormatInt(*pw.Allow, 10)
			}
			if pw.Deny != nil {
				req.PermissionOverwrites[i].Deny = strconv.FormatInt(*pw.Deny, 10)
			}
		}
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
		if isDiscordNotFound(err) {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete channel")
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// Nothing to disconnect for Discord API client
	return nil
}
