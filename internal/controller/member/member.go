package member

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-discord/apis/member/v1alpha1"
	"github.com/rossigee/provider-discord/apis/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotMember    = "managed resource is not a Member custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

// Setup adds a controller that reconciles Member managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.MemberGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.MemberGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Member{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Member)
	if !ok {
		return nil, errors.New(errNotMember)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Get provider config reference from the managed resource's ResourceSpec
	var pcRef *xpv1.Reference

	// Type assert to extract the ProviderConfigReference from the managed resource
	switch mr := mg.(type) {
	case interface{ GetProviderConfigReference() *xpv1.Reference }:
		pcRef = mr.GetProviderConfigReference()
	default:
		return nil, errors.New(errGetPC)
	}

	if pcRef == nil {
		return nil, errors.New(errGetPC)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: pcRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	// Extract credentials from the provider config
	credentials := discordclient.ProviderCredentials{
		Source:                      discordclient.CredentialsSourceSecret,
		CommonCredentialSelectors:   pc.Spec.Credentials.CommonCredentialSelectors,
	}
	token, err := credentials.Extract(ctx, c.kube)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	// Create Discord client
	discordClient := discordclient.NewDiscordClient(token)

	return &external{discord: discordClient}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	discord discordclient.MemberClient
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Member)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotMember)
	}

	// Get external name (Discord User ID)
	userID := meta.GetExternalName(cr)
	if userID == "" {
		// Check if we have an ID in status
		if cr.Status.AtProvider.User != nil && cr.Status.AtProvider.User.ID != "" {
			// Set external name from status
			meta.SetExternalName(cr, cr.Status.AtProvider.User.ID)
			userID = cr.Status.AtProvider.User.ID
		} else {
			// No external resource exists
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
	}

	// Get the member from Discord
	member, err := e.discord.GetGuildMember(ctx, cr.Spec.ForProvider.GuildID, userID)
	if err != nil {
		if err.Error() == "member not found" {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get member")
	}

	// Update status - populate user information
	if member.User != nil {
		cr.Status.AtProvider.User = &v1alpha1.DiscordUser{
			ID:            member.User.ID,
			Username:      member.User.Username,
			Discriminator: member.User.Discriminator,
			Avatar:        member.User.Avatar,
		}
		// Also populate top-level fields for backward compatibility
		cr.Status.AtProvider.ID = member.User.ID
		cr.Status.AtProvider.Username = member.User.Username
		cr.Status.AtProvider.Discriminator = member.User.Discriminator
	}
	cr.Status.AtProvider.Nick = member.Nick
	cr.Status.AtProvider.Avatar = member.Avatar
	cr.Status.AtProvider.Roles = member.Roles
	cr.Status.AtProvider.JoinedAt = member.JoinedAt
	cr.Status.AtProvider.PremiumSince = member.PremiumSince
	cr.Status.AtProvider.Deaf = &member.Deaf
	cr.Status.AtProvider.Mute = &member.Mute
	cr.Status.AtProvider.Flags = &member.Flags
	cr.Status.AtProvider.Pending = member.Pending
	cr.Status.AtProvider.Permissions = member.Permissions
	cr.Status.AtProvider.CommunicationDisabledUntil = member.CommunicationDisabledUntil

	// Check if update is needed - compare nickname and roles
	needsUpdate := (cr.Spec.ForProvider.Nick != nil && (member.Nick == nil || *cr.Spec.ForProvider.Nick != *member.Nick)) ||
		(cr.Spec.ForProvider.Nick == nil && member.Nick != nil) ||
		len(cr.Spec.ForProvider.Roles) != len(member.Roles)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        !needsUpdate,
		ResourceLateInitialized: false,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, ok := mg.(*v1alpha1.Member)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotMember)
	}

	// For members, we don't actually "create" them - users join guilds through invites
	// This operation would typically add an existing user to the guild via OAuth2
	// For now, we'll return an error indicating this operation is not supported
	return managed.ExternalCreation{}, errors.New("creating members is not supported - members join through invites or OAuth2")
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Member)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotMember)
	}

	userID := meta.GetExternalName(cr)
	if userID == "" {
		return managed.ExternalUpdate{}, errors.New("cannot update member without external name")
	}

	// Build modify request
	req := &discordclient.ModifyGuildMemberRequest{}
	
	if cr.Spec.ForProvider.Nick != nil {
		req.Nick = cr.Spec.ForProvider.Nick
	}
	
	if len(cr.Spec.ForProvider.Roles) > 0 {
		req.Roles = cr.Spec.ForProvider.Roles
	}
	
	if cr.Spec.ForProvider.Mute != nil {
		req.Mute = cr.Spec.ForProvider.Mute
	}
	
	if cr.Spec.ForProvider.Deaf != nil {
		req.Deaf = cr.Spec.ForProvider.Deaf
	}
	
	if cr.Spec.ForProvider.ChannelID != nil {
		req.ChannelID = cr.Spec.ForProvider.ChannelID
	}
	
	if cr.Spec.ForProvider.CommunicationDisabledUntil != nil {
		req.CommunicationDisabledUntil = cr.Spec.ForProvider.CommunicationDisabledUntil
	}

	_, err := e.discord.ModifyGuildMember(ctx, cr.Spec.ForProvider.GuildID, userID, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update member")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Member)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotMember)
	}

	userID := meta.GetExternalName(cr)
	if userID == "" {
		// No external resource to delete
		return managed.ExternalDelete{}, nil
	}

	err := e.discord.RemoveGuildMember(ctx, cr.Spec.ForProvider.GuildID, userID)
	if err != nil {
		if err.Error() == "member not found" {
			// Member already removed
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to remove member")
	}

	return managed.ExternalDelete{}, nil
}