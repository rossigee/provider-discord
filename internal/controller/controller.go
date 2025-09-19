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
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"github.com/rossigee/provider-discord/internal/clients"
	"github.com/rossigee/provider-discord/internal/controller/application"
	"github.com/rossigee/provider-discord/internal/controller/channel"
	"github.com/rossigee/provider-discord/internal/controller/config"
	"github.com/rossigee/provider-discord/internal/controller/guild"
	"github.com/rossigee/provider-discord/internal/controller/integration"
	"github.com/rossigee/provider-discord/internal/controller/invite"
	"github.com/rossigee/provider-discord/internal/controller/member"
	"github.com/rossigee/provider-discord/internal/controller/role"
	"github.com/rossigee/provider-discord/internal/controller/user"
	"github.com/rossigee/provider-discord/internal/controller/webhook"
	"github.com/rossigee/provider-discord/internal/metrics"
)

// Setup creates all Discord controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return SetupWithMetrics(mgr, o, nil)
}

// SetupWithMetrics creates all Discord controllers with metrics support and adds them to
// the supplied manager.
func SetupWithMetrics(mgr ctrl.Manager, o controller.Options, metricsRecorder *metrics.MetricsRecorder) error {
	// Create a Discord client factory function that includes metrics
	newDiscordClientFn := func(token string) *clients.DiscordClient {
		return clients.NewDiscordClientWithMetrics(token, metricsRecorder)
	}

	// For now, just update the controllers that use the managed approach (using function pointers)
	// Start with the channel controller as a test
	if err := channel.SetupWithClient(mgr, o, newDiscordClientFn); err != nil {
		return err
	}

	// Use regular setup for other controllers for now
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
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
	return nil
}