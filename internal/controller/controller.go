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

package controller

import (
	"context"
	"os"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/rossigee/provider-discord/internal/clients"
	"github.com/rossigee/provider-discord/internal/controller/application"
	"github.com/rossigee/provider-discord/internal/controller/channel"
	"github.com/rossigee/provider-discord/internal/controller/deduplication"
	"github.com/rossigee/provider-discord/internal/controller/garbagecollection"
	"github.com/rossigee/provider-discord/internal/controller/guild"
	"github.com/rossigee/provider-discord/internal/controller/integration"
	"github.com/rossigee/provider-discord/internal/controller/invite"
	"github.com/rossigee/provider-discord/internal/controller/member"
	"github.com/rossigee/provider-discord/internal/controller/providerconfig"
	"github.com/rossigee/provider-discord/internal/controller/role"
	"github.com/rossigee/provider-discord/internal/controller/user"
	"github.com/rossigee/provider-discord/internal/controller/webhook"
	"github.com/rossigee/provider-discord/internal/metrics"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Setup creates all Discord controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return SetupWithMetrics(mgr, o, nil)
}

// SetupWithMetrics creates all Discord controllers with metrics support and adds them to
// the supplied manager.
func SetupWithMetrics(mgr ctrl.Manager, o controller.Options, metricsRecorder *metrics.MetricsRecorder) error {
	// Self-managed RBAC (stable names, no more per-revision pinning)
	if err := setupRBAC(mgr.GetClient(), o.Logger); err != nil {
		o.Logger.Info("RBAC setup warning (may be transient)", "error", err)
	}

	if err := providerconfig.Setup(mgr); err != nil {
		return err
	}

	// Setup all controllers using regular Setup functions
	// The metrics will be integrated at the client level
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		// NOTE: ProviderConfig controller removed - crossplane-runtime handles this automatically
		// config.Setup,
		channel.Setup,
		guild.Setup,
		role.Setup,
		webhook.Setup,
		invite.Setup,
		member.Setup,
		user.Setup,
		application.Setup,
		integration.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}

	// Setup deduplication controller (watches ProviderConfig annotations)
	if err := deduplication.Setup(mgr); err != nil {
		return err
	}

	// Setup garbage collection controller (autonomous cleanup management)
	gc := &garbagecollection.ProviderConfigReconciler{}
	if err := gc.SetupWithManager(mgr); err != nil {
		return err
	}

	// Set the global metrics recorder for client use
	if metricsRecorder != nil {
		clients.SetGlobalMetricsRecorder(metricsRecorder)
	}

	return nil
}

// setupRBAC ensures the provider's stable RBAC objects (system + binding + aggregates).
func setupRBAC(c client.Client, l logging.Logger) error {
	ctx := context.Background()

	rules := []rbacv1.PolicyRule{
		{APIGroups: []string{"application.discord.crossplane.io"}, Resources: []string{"applications", "applications/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"channel.discord.crossplane.io"}, Resources: []string{"channels", "channels/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"deduplication.discord.crossplane.io"}, Resources: []string{"deduplications", "deduplications/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"guild.discord.crossplane.io"}, Resources: []string{"guilds", "guilds/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"integration.discord.crossplane.io"}, Resources: []string{"integrations", "integrations/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"invite.discord.crossplane.io"}, Resources: []string{"invites", "invites/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"member.discord.crossplane.io"}, Resources: []string{"members", "members/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"role.discord.crossplane.io"}, Resources: []string{"roles", "roles/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"user.discord.crossplane.io"}, Resources: []string{"users", "users/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"webhook.discord.crossplane.io"}, Resources: []string{"webhooks", "webhooks/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"discord.crossplane.io"}, Resources: []string{"providerconfigs", "providerconfigs/status", "providerconfigusages", "providerconfigusages/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{
			APIGroups: []string{"application.discord.crossplane.io", "channel.discord.crossplane.io", "deduplication.discord.crossplane.io", "guild.discord.crossplane.io", "integration.discord.crossplane.io", "invite.discord.crossplane.io", "member.discord.crossplane.io", "role.discord.crossplane.io", "user.discord.crossplane.io", "webhook.discord.crossplane.io", "discord.crossplane.io"},
			Resources: []string{"*/finalizers"},
			Verbs:     []string{"update"},
		},
		{APIGroups: []string{"", "coordination.k8s.io"}, Resources: []string{"secrets", "configmaps", "events", "leases"}, Verbs: []string{"*"}},
	}

	system := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "crossplane:provider:provider-discord:system",
			Labels: map[string]string{"rbac.crossplane.io/system": "provider-discord"},
		},
		Rules: rules,
	}
	if err := c.Create(ctx, system); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	if err := c.Update(ctx, system); err != nil {
		l.Info("system role update", "err", err)
	}

	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "crossplane:provider:provider-discord:system"},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "crossplane:provider:provider-discord:system"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: os.Getenv("REVISION_NAME"), Namespace: "crossplane-system"}},
	}
	if err := c.Create(ctx, binding); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	if err := c.Update(ctx, binding); err != nil {
		l.Info("system binding update", "err", err)
	}

	// aggregates best-effort
	edit := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "crossplane:provider:provider-discord:aggregate-to-edit",
			Labels: map[string]string{
				"rbac.crossplane.io/aggregate-to-edit": "true", "rbac.crossplane.io/aggregate-to-admin": "true",
				"rbac.crossplane.io/aggregate-to-crossplane": "true", "rbac.crossplane.io/system": "provider-discord",
			},
		},
		Rules: withVerbs(rules, []string{"*"}),
	}
	if err := c.Create(ctx, edit); err != nil && !errors.IsAlreadyExists(err) {
		l.Info("aggregate-to-edit create warning (non-fatal)", "err", err)
	}
	_ = c.Update(ctx, edit)

	view := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "crossplane:provider:provider-discord:aggregate-to-view",
			Labels: map[string]string{"rbac.crossplane.io/aggregate-to-view": "true", "rbac.crossplane.io/system": "provider-discord"},
		},
		Rules: withVerbs(rules, []string{"get", "list", "watch"}),
	}
	if err := c.Create(ctx, view); err != nil && !errors.IsAlreadyExists(err) {
		l.Info("aggregate-to-view create warning (non-fatal)", "err", err)
	}
	_ = c.Update(ctx, view)

	l.Info("provider self-managed RBAC roles ensured")
	return nil
}

func withVerbs(r []rbacv1.PolicyRule, verbs []string) []rbacv1.PolicyRule {
	out := make([]rbacv1.PolicyRule, len(r))
	for i := range r {
		out[i] = r[i]
		out[i].Verbs = verbs
	}
	return out
}
