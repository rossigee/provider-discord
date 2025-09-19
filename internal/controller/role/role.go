package role

import (
	"context"
	"fmt"

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

	"github.com/rossigee/provider-discord/apis/role/v1alpha1"
	"github.com/rossigee/provider-discord/apis/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
)

const (
	errNotRole      = "managed resource is not a Role custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	// errNewClient removed - unused
)

// Setup adds a controller that reconciles Role managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RoleGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RoleGroupVersionKind),
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
		For(&v1alpha1.Role{}).
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
	_, ok := mg.(*v1alpha1.Role)
	if !ok {
		return nil, errors.New(errNotRole)
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
	discord discordclient.RoleClient
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Role)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRole)
	}

	// Get external name (Discord Role ID)
	roleID := meta.GetExternalName(cr)
	if roleID == "" {
		// Check if we have an ID in status
		if cr.Status.AtProvider.ID != "" {
			// Set external name from status
			meta.SetExternalName(cr, cr.Status.AtProvider.ID)
			roleID = cr.Status.AtProvider.ID
		} else {
			// No external resource exists
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
	}

	// Get the role from Discord
	role, err := e.discord.GetRole(ctx, cr.Spec.ForProvider.GuildID, roleID)
	if err != nil {
		if err.Error() == "role not found" {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get role")
	}

	// Update status
	cr.Status.AtProvider.ID = role.ID
	cr.Status.AtProvider.Managed = role.Managed

	// Check if update is needed
	needsUpdate := role.Name != cr.Spec.ForProvider.Name ||
		(cr.Spec.ForProvider.Color != nil && role.Color != *cr.Spec.ForProvider.Color)
	if cr.Spec.ForProvider.Hoist != nil && role.Hoist != *cr.Spec.ForProvider.Hoist {
		needsUpdate = true
	}
	if cr.Spec.ForProvider.Mentionable != nil && role.Mentionable != *cr.Spec.ForProvider.Mentionable {
		needsUpdate = true
	}
	if cr.Spec.ForProvider.Permissions != nil && role.Permissions != *cr.Spec.ForProvider.Permissions {
		needsUpdate = true
	}
	if cr.Spec.ForProvider.Position != nil && role.Position != *cr.Spec.ForProvider.Position {
		needsUpdate = true
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: !needsUpdate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Role)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRole)
	}

	// Create role request
	req := discordclient.CreateRoleRequest{
		Name:        cr.Spec.ForProvider.Name,
		Permissions: cr.Spec.ForProvider.Permissions,
		Color:       cr.Spec.ForProvider.Color,
		Hoist:       cr.Spec.ForProvider.Hoist,
		Mentionable: cr.Spec.ForProvider.Mentionable,
	}

	// Create the role
	role, err := e.discord.CreateRole(ctx, cr.Spec.ForProvider.GuildID, req)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create role")
	}

	// Set external name to the Discord role ID
	meta.SetExternalName(cr, role.ID)
	cr.Status.AtProvider.ID = role.ID
	cr.Status.AtProvider.Managed = role.Managed

	// Handle position separately if specified
	if cr.Spec.ForProvider.Position != nil {
		modifyReq := discordclient.ModifyRoleRequest{
			Position: cr.Spec.ForProvider.Position,
		}
		_, err = e.discord.ModifyRole(ctx, cr.Spec.ForProvider.GuildID, role.ID, modifyReq)
		if err != nil {
			// Log error but don't fail creation
			fmt.Printf("Warning: failed to set role position: %v\n", err)
		}
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Role)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRole)
	}

	roleID := meta.GetExternalName(cr)
	if roleID == "" {
		return managed.ExternalUpdate{}, errors.New("external name (role ID) not set")
	}

	// Build update request
	req := discordclient.ModifyRoleRequest{
		Name:        &cr.Spec.ForProvider.Name,
		Permissions: cr.Spec.ForProvider.Permissions,
		Color:       cr.Spec.ForProvider.Color,
		Hoist:       cr.Spec.ForProvider.Hoist,
		Position:    cr.Spec.ForProvider.Position,
		Mentionable: cr.Spec.ForProvider.Mentionable,
	}

	// Update the role
	_, err := e.discord.ModifyRole(ctx, cr.Spec.ForProvider.GuildID, roleID, req)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update role")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Role)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRole)
	}

	roleID := meta.GetExternalName(cr)
	if roleID == "" {
		// Nothing to delete if we don't have an ID
		return managed.ExternalDelete{}, nil
	}

	// Delete the role
	err := e.discord.DeleteRole(ctx, cr.Spec.ForProvider.GuildID, roleID)
	if err != nil {
		// If role is already gone, don't error
		if err.Error() == "role not found" {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete role")
	}

	return managed.ExternalDelete{}, nil
}