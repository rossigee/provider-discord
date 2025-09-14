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

package invite

import (
	"context"
	"regexp"
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

	invitev1alpha1 "github.com/rossigee/provider-discord/apis/invite/v1alpha1"
	"github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotInvite    = "managed resource is not an Invite custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

var (
	// Discord invite codes are typically 6-12 character alphanumeric strings
	// Examples: "abc123", "xyz789", "discord", "general"
	discordInviteCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9]{3,12}$`)
)

// isValidDiscordInviteCode checks if the provided string looks like a Discord invite code
func isValidDiscordInviteCode(code string) bool {
	// Resource names like "general-channel-invite" should fail this validation
	// Real Discord invite codes are short alphanumeric strings without dashes/underscores
	return discordInviteCodeRegex.MatchString(code)
}

// Setup adds a controller that reconciles Invite managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(invitev1alpha1.InviteGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(invitev1alpha1.InviteGroupVersionKind),
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
		For(&invitev1alpha1.Invite{}).
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
	cr, ok := mg.(*invitev1alpha1.Invite)
	if !ok {
		return nil, errors.New(errNotInvite)
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
	service clients.InviteClient
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*invitev1alpha1.Invite)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotInvite)
	}

	externalName := meta.GetExternalName(cr)
	
	// If external-name is empty or not a valid Discord invite code, this is a new resource to be created
	// Crossplane runtime defaults external-name to metadata.name for new resources
	if externalName == "" || !isValidDiscordInviteCode(externalName) {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// If we have a valid external name (Discord invite code), try to get by code
	invite, err := c.service.GetInvite(ctx, externalName)
	if err != nil {
		// If invite not found by code, assume it needs to be created
		// This handles cases where external-name was set but invite doesn't exist
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	if invite == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Parse expiration time if present
	var expiresAt *metav1.Time
	if invite.ExpiresAt != nil {
		if parsedTime, err := time.Parse(time.RFC3339, *invite.ExpiresAt); err == nil {
			expiresAt = &metav1.Time{Time: parsedTime}
		}
	}

	// Parse created time
	var createdAt *metav1.Time
	if parsedTime, err := time.Parse(time.RFC3339, invite.CreatedAt); err == nil {
		createdAt = &metav1.Time{Time: parsedTime}
	}

	// Update status with observed values
	cr.Status.AtProvider = invitev1alpha1.InviteObservation{
		Code:                     invite.Code,
		GuildID:                  getStringFromGuild(invite.Guild),
		ChannelID:                getStringFromChannel(invite.Channel),
		InviterID:                getStringFromUser(invite.Inviter),
		TargetType:               invite.TargetType,
		TargetUserID:             getStringPtrFromUser(invite.TargetUser),
		TargetApplicationID:      getStringPtrFromApplication(invite.TargetApplication),
		ApproximatePresenceCount: invite.ApproximatePresenceCount,
		ApproximateMemberCount:   invite.ApproximateMemberCount,
		ExpiresAt:                expiresAt,
		CreatedAt:                createdAt,
		Uses:                     invite.Uses,
		MaxAge:                   invite.MaxAge,
		MaxUses:                  invite.MaxUses,
		Temporary:                invite.Temporary,
	}

	// Store invite URL in connection secret
	connectionDetails := managed.ConnectionDetails{}
	if invite.Code != "" {
		inviteURL := "https://discord.gg/" + invite.Code
		connectionDetails["url"] = []byte(inviteURL)
	}

	// Invites cannot be updated, so always up to date if it exists
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  true,
		ConnectionDetails: connectionDetails,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*invitev1alpha1.Invite)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotInvite)
	}

	cr.SetConditions(xpv1.Creating())

	req := &clients.CreateInviteRequest{
		MaxAge:    cr.Spec.ForProvider.MaxAge,
		MaxUses:   cr.Spec.ForProvider.MaxUses,
		Temporary: cr.Spec.ForProvider.Temporary,
		Unique:    cr.Spec.ForProvider.Unique,
	}

	invite, err := c.service.CreateChannelInvite(ctx, cr.Spec.ForProvider.ChannelID, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create invite")
	}

	meta.SetExternalName(cr, invite.Code)

	// Store invite URL in connection secret
	connectionDetails := managed.ConnectionDetails{}
	if invite.Code != "" {
		inviteURL := "https://discord.gg/" + invite.Code
		connectionDetails["url"] = []byte(inviteURL)
	}

	return managed.ExternalCreation{
		ConnectionDetails: connectionDetails,
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// Discord invites cannot be updated after creation
	// This is consistent with Discord API behavior
	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*invitev1alpha1.Invite)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotInvite)
	}

	cr.SetConditions(xpv1.Deleting())

	err := c.service.DeleteInvite(ctx, meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete invite")
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// Nothing to disconnect for Discord API client
	return nil
}

// Helper functions to safely extract IDs from nested structs

func getStringFromGuild(guild *clients.Guild) string {
	if guild != nil {
		return guild.ID
	}
	return ""
}

func getStringFromChannel(channel *clients.Channel) string {
	if channel != nil {
		return channel.ID
	}
	return ""
}

func getStringFromUser(user *clients.User) string {
	if user != nil {
		return user.ID
	}
	return ""
}

func getStringPtrFromUser(user *clients.User) *string {
	if user != nil {
		return &user.ID
	}
	return nil
}

func getStringPtrFromApplication(app *clients.Application) *string {
	if app != nil {
		return &app.ID
	}
	return nil
}