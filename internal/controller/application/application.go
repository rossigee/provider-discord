package application

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/rossigee/provider-discord/apis/application/v1alpha1"
	"github.com/rossigee/provider-discord/apis/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotApplication = "managed resource is not an Application custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
)

// Setup adds a controller that reconciles Application managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ApplicationGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ApplicationGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Application{}).
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
	cr, ok := mg.(*v1alpha1.Application)
	if !ok {
		return nil, errors.New(errNotApplication)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
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
	discord discordclient.ApplicationClient
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotApplication)
	}

	// Determine which application to get - either @me or a specific application ID
	var app *discordclient.DiscordApplication
	var err error

	if cr.Spec.ForProvider.ApplicationID == "@me" {
		app, err = e.discord.GetCurrentApplication(ctx)
	} else {
		app, err = e.discord.GetApplication(ctx, cr.Spec.ForProvider.ApplicationID)
	}

	if err != nil {
		if err.Error() == "application not found" {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get application")
	}

	// Set external name to the actual application ID
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, app.ID)
	}

	// Update status
	cr.Status.AtProvider.ID = app.ID
	cr.Status.AtProvider.Name = app.Name
	cr.Status.AtProvider.Icon = app.Icon
	cr.Status.AtProvider.Description = app.Description
	cr.Status.AtProvider.RPCOrigins = app.RPCOrigins
	cr.Status.AtProvider.BotPublic = app.BotPublic
	cr.Status.AtProvider.BotRequireCodeGrant = app.BotRequireCodeGrant
	if app.Bot != nil {
		if botID, ok := app.Bot["id"].(string); ok {
			cr.Status.AtProvider.BotUserID = &botID
		}
	}
	cr.Status.AtProvider.TermsOfServiceURL = app.TermsOfServiceURL
	cr.Status.AtProvider.PrivacyPolicyURL = app.PrivacyPolicyURL
	if app.Owner != nil {
		if ownerID, ok := app.Owner["id"].(string); ok {
			cr.Status.AtProvider.OwnerID = &ownerID
		}
	}
	cr.Status.AtProvider.Summary = app.Summary
	cr.Status.AtProvider.VerifyKey = app.VerifyKey
	if app.Team != nil {
		if teamID, ok := app.Team["id"].(string); ok {
			cr.Status.AtProvider.TeamID = &teamID
		}
	}
	cr.Status.AtProvider.GuildID = app.GuildID
	cr.Status.AtProvider.PrimarySkuID = app.PrimarySkuID
	cr.Status.AtProvider.Slug = app.Slug
	cr.Status.AtProvider.CoverImage = app.CoverImage
	cr.Status.AtProvider.Flags = app.Flags
	cr.Status.AtProvider.ApproximateGuildCount = app.ApproximateGuildCount
	cr.Status.AtProvider.RedirectURIs = app.RedirectURIs
	cr.Status.AtProvider.InteractionsEndpointURL = app.InteractionsEndpointURL
	cr.Status.AtProvider.RoleConnectionsVerificationURL = app.RoleConnectionsVerificationURL
	cr.Status.AtProvider.Tags = app.Tags
	if app.InstallParams != nil {
		if scopes, ok := app.InstallParams["scopes"].([]interface{}); ok {
			installScopes := make([]string, len(scopes))
			for i, scope := range scopes {
				if scopeStr, ok := scope.(string); ok {
					installScopes[i] = scopeStr
				}
			}
			cr.Status.AtProvider.InstallParamsScopes = installScopes
		}
		if permissions, ok := app.InstallParams["permissions"].(string); ok {
			cr.Status.AtProvider.InstallParamsPermissions = &permissions
		}
	}
	cr.Status.AtProvider.CustomInstallURL = app.CustomInstallURL

	// Check if update is needed (only for current application)
	needsUpdate := false
	if cr.Spec.ForProvider.ApplicationID == "@me" {
		if cr.Spec.ForProvider.Name != nil && *cr.Spec.ForProvider.Name != app.Name {
			needsUpdate = true
		}
		if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != app.Description {
			needsUpdate = true
		}
		// Can only update current application, not arbitrary applications
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        !needsUpdate,
		ResourceLateInitialized: false,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotApplication)
	}

	// Applications can't be "created" via the API - they're created through the Discord developer portal
	// This resource is primarily for reading and updating application information
	return managed.ExternalCreation{}, errors.New("applications cannot be created via the API")
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotApplication)
	}

	// Only current application (@me) can be updated
	if cr.Spec.ForProvider.ApplicationID != "@me" {
		return managed.ExternalUpdate{}, errors.New("only current application (@me) can be updated")
	}

	// Build modify current application request
	req := &discordclient.ModifyCurrentApplicationRequest{}
	
	if cr.Spec.ForProvider.Name != nil {
		req.Name = cr.Spec.ForProvider.Name
	}
	
	if cr.Spec.ForProvider.Description != nil {
		req.Description = cr.Spec.ForProvider.Description
	}
	
	if cr.Spec.ForProvider.Icon != nil {
		req.Icon = cr.Spec.ForProvider.Icon
	}
	
	if cr.Spec.ForProvider.CoverImage != nil {
		req.CoverImage = cr.Spec.ForProvider.CoverImage
	}
	
	if len(cr.Spec.ForProvider.RPCOrigins) > 0 {
		req.RPCOrigins = cr.Spec.ForProvider.RPCOrigins
	}
	
	if cr.Spec.ForProvider.BotPublic != nil {
		req.BotPublic = cr.Spec.ForProvider.BotPublic
	}
	
	if cr.Spec.ForProvider.BotRequireCodeGrant != nil {
		req.BotRequireCodeGrant = cr.Spec.ForProvider.BotRequireCodeGrant
	}
	
	if cr.Spec.ForProvider.TermsOfServiceURL != nil {
		req.TermsOfServiceURL = cr.Spec.ForProvider.TermsOfServiceURL
	}
	
	if cr.Spec.ForProvider.PrivacyPolicyURL != nil {
		req.PrivacyPolicyURL = cr.Spec.ForProvider.PrivacyPolicyURL
	}
	
	if cr.Spec.ForProvider.CustomInstallURL != nil {
		req.CustomInstallURL = cr.Spec.ForProvider.CustomInstallURL
	}
	
	if len(cr.Spec.ForProvider.Tags) > 0 {
		req.Tags = cr.Spec.ForProvider.Tags
	}

	_, err := e.discord.ModifyCurrentApplication(ctx, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update current application")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotApplication)
	}

	// Applications can't be deleted via the API - this is a read-only resource
	// We just remove our tracking of it
	return managed.ExternalDelete{}, nil
}