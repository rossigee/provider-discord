package integration

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

	integrationv1alpha1 "github.com/rossigee/provider-discord/apis/integration/v1alpha1"
	v1alpha1 "github.com/rossigee/provider-discord/apis/v1alpha1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotIntegration = "managed resource is not an Integration custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
)

// Setup adds a controller that reconciles Integration managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(integrationv1alpha1.IntegrationGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(integrationv1alpha1.IntegrationGroupVersionKind),
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
		For(&integrationv1alpha1.Integration{}).
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
	_, ok := mg.(*integrationv1alpha1.Integration)
	if !ok {
		return nil, errors.New(errNotIntegration)
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

	pc := &v1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: pcRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	// Extract credentials from the provider config
	credentials := discordclient.ProviderCredentials{
		Source:                    discordclient.CredentialsSourceSecret,
		CommonCredentialSelectors: pc.Spec.Credentials.CommonCredentialSelectors,
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
	discord discordclient.IntegrationClient
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*integrationv1alpha1.Integration)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIntegration)
	}

	// Get all integrations for the guild
	integrations, err := e.discord.GetGuildIntegrations(ctx, cr.Spec.ForProvider.GuildID)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get guild integrations")
	}

	// Find the specific integration by ID
	var foundIntegration *discordclient.GuildIntegration
	for _, integration := range integrations {
		if integration.ID == cr.Spec.ForProvider.IntegrationID {
			foundIntegration = &integration
			break
		}
	}

	if foundIntegration == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Set external name to the integration ID
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, foundIntegration.ID)
	}

	// Update status
	cr.Status.AtProvider.ID = foundIntegration.ID
	cr.Status.AtProvider.Name = foundIntegration.Name
	cr.Status.AtProvider.Type = foundIntegration.Type
	cr.Status.AtProvider.Enabled = foundIntegration.Enabled
	cr.Status.AtProvider.Syncing = foundIntegration.Syncing
	cr.Status.AtProvider.RoleID = foundIntegration.RoleID
	cr.Status.AtProvider.EnableEmoticons = foundIntegration.EnableEmoticons
	cr.Status.AtProvider.ExpireBehavior = foundIntegration.ExpireBehavior
	cr.Status.AtProvider.ExpireGracePeriod = foundIntegration.ExpireGracePeriod
	if foundIntegration.User != nil {
		if userID, ok := foundIntegration.User["id"].(string); ok {
			cr.Status.AtProvider.UserID = &userID
		}
	}
	if foundIntegration.Account != nil {
		if accountID, ok := foundIntegration.Account["id"].(string); ok {
			cr.Status.AtProvider.AccountID = &accountID
		}
		if accountName, ok := foundIntegration.Account["name"].(string); ok {
			cr.Status.AtProvider.AccountName = &accountName
		}
	}
	cr.Status.AtProvider.SyncedAt = foundIntegration.SyncedAt
	cr.Status.AtProvider.SubscriberCount = foundIntegration.SubscriberCount
	cr.Status.AtProvider.Revoked = foundIntegration.Revoked
	if foundIntegration.Application != nil {
		if appID, ok := foundIntegration.Application["id"].(string); ok {
			cr.Status.AtProvider.ApplicationID = &appID
		}
	}
	cr.Status.AtProvider.Scopes = foundIntegration.Scopes

	// Integrations are typically managed externally and can't be updated via API
	// This resource is primarily for observing integration state
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        true, // Since we can't modify integrations
		ResourceLateInitialized: false,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, ok := mg.(*integrationv1alpha1.Integration)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIntegration)
	}

	// Integrations can't be "created" via the Discord API
	// They are created through external OAuth2 flows (e.g., connecting Twitch, YouTube, etc.)
	// This resource is for managing existing integrations, not creating new ones
	return managed.ExternalCreation{}, errors.New("integrations cannot be created via the API - they are created through OAuth2 flows")
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*integrationv1alpha1.Integration)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIntegration)
	}

	// Discord API doesn't provide endpoints to modify integrations
	// Integrations are typically managed through their respective platforms
	return managed.ExternalUpdate{}, errors.New("integrations cannot be modified via the API")
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*integrationv1alpha1.Integration)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotIntegration)
	}

	integrationID := meta.GetExternalName(cr)
	if integrationID == "" {
		integrationID = cr.Spec.ForProvider.IntegrationID
	}

	if integrationID == "" {
		// No external resource to delete
		return managed.ExternalDelete{}, nil
	}

	err := e.discord.DeleteGuildIntegration(ctx, cr.Spec.ForProvider.GuildID, integrationID)
	if err != nil {
		if err.Error() == "integration not found" {
			// Integration already removed
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete integration")
	}

	return managed.ExternalDelete{}, nil
}
