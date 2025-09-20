package user

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

	"github.com/rossigee/provider-discord/apis/user/v1alpha1"
	"github.com/rossigee/provider-discord/apis/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotUser      = "managed resource is not a User custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.UserGroupVersionKind),
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
		For(&v1alpha1.User{}).
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
	_, ok := mg.(*v1alpha1.User)
	if !ok {
		return nil, errors.New(errNotUser)
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
	discord discordclient.UserClient
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	// Determine which user to get - either @me or a specific user ID
	var user *discordclient.DiscordUser
	var err error

	if cr.Spec.ForProvider.UserID == "@me" {
		user, err = e.discord.GetCurrentUser(ctx)
	} else {
		user, err = e.discord.GetUser(ctx, cr.Spec.ForProvider.UserID)
	}

	if err != nil {
		if err.Error() == "user not found" {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get user")
	}

	// Set external name to the actual user ID
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, user.ID)
	}

	// Update status
	cr.Status.AtProvider.ID = user.ID
	cr.Status.AtProvider.Username = user.Username
	cr.Status.AtProvider.Discriminator = user.Discriminator
	cr.Status.AtProvider.GlobalName = user.GlobalName
	cr.Status.AtProvider.Avatar = user.Avatar
	cr.Status.AtProvider.Bot = user.Bot
	cr.Status.AtProvider.System = user.System
	cr.Status.AtProvider.MFAEnabled = user.MFAEnabled
	cr.Status.AtProvider.Banner = user.Banner
	cr.Status.AtProvider.AccentColor = user.AccentColor
	cr.Status.AtProvider.Locale = user.Locale
	cr.Status.AtProvider.Verified = user.Verified
	cr.Status.AtProvider.Email = user.Email
	cr.Status.AtProvider.Flags = user.Flags
	cr.Status.AtProvider.PremiumType = user.PremiumType
	cr.Status.AtProvider.PublicFlags = user.PublicFlags
	if user.AvatarDecoration != nil {
		// Convert map to string representation if needed
		if decorationData, exists := user.AvatarDecoration["asset"]; exists {
			if asset, ok := decorationData.(string); ok {
				cr.Status.AtProvider.AvatarDecorationData = &asset
			}
		}
	}

	// Check if update is needed (only for current user)
	needsUpdate := false
	if cr.Spec.ForProvider.UserID == "@me" {
		if cr.Spec.ForProvider.Username != nil && *cr.Spec.ForProvider.Username != user.Username {
			needsUpdate = true
		}
		// Can only update current user, not arbitrary users
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        !needsUpdate,
		ResourceLateInitialized: false,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	// Users can't be "created" via the API - they exist independently
	// This resource is primarily for reading user information
	return managed.ExternalCreation{}, errors.New("users cannot be created via the API")
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	// Only current user (@me) can be updated
	if cr.Spec.ForProvider.UserID != "@me" {
		return managed.ExternalUpdate{}, errors.New("only current user (@me) can be updated")
	}

	// Build modify current user request
	req := &discordclient.ModifyCurrentUserRequest{}

	if cr.Spec.ForProvider.Username != nil {
		req.Username = cr.Spec.ForProvider.Username
	}

	if cr.Spec.ForProvider.Avatar != nil {
		req.Avatar = cr.Spec.ForProvider.Avatar
	}

	if cr.Spec.ForProvider.Banner != nil {
		req.Banner = cr.Spec.ForProvider.Banner
	}

	_, err := e.discord.ModifyCurrentUser(ctx, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update current user")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	// Users can't be deleted via the API - this is a read-only resource
	// We just remove our tracking of it
	return managed.ExternalDelete{}, nil
}
