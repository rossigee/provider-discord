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

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/alecthomas/kingpin/v2"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/rossigee/provider-discord/apis"
	applicationv1beta1 "github.com/rossigee/provider-discord/apis/application/v1beta1"
	channelv1beta1 "github.com/rossigee/provider-discord/apis/channel/v1beta1"
	guildv1beta1 "github.com/rossigee/provider-discord/apis/guild/v1beta1"
	integrationv1beta1 "github.com/rossigee/provider-discord/apis/integration/v1beta1"
	invitev1beta1 "github.com/rossigee/provider-discord/apis/invite/v1beta1"
	memberv1beta1 "github.com/rossigee/provider-discord/apis/member/v1beta1"
	rolev1beta1 "github.com/rossigee/provider-discord/apis/role/v1beta1"
	userv1beta1 "github.com/rossigee/provider-discord/apis/user/v1beta1"
	webhookv1beta1 "github.com/rossigee/provider-discord/apis/webhook/v1beta1"
	"github.com/rossigee/provider-discord/internal/controller"
	"github.com/rossigee/provider-discord/internal/features"
	"github.com/rossigee/provider-discord/internal/metrics"
	"github.com/rossigee/provider-discord/internal/tracing"
	"github.com/rossigee/provider-discord/internal/version"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	sigzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	metricserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func main() {
	var (
		app                      = kingpin.New(filepath.Base(os.Args[0]), "Discord support for Crossplane.").DefaultEnvars()
		debug                    = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		leaderElection           = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		leaderElectionNS         = app.Flag("leader-election-namespace", "Namespace to use for leader election.").Default("crossplane-system").OverrideDefaultFromEnvar("LEADER_ELECTION_NAMESPACE").String()
		pollInterval             = app.Flag("poll", "How often individual resources will be checked for drift from the desired state (default 5m to respect Discord's strict global rate limits; Discord's per-route limits are ~5 req/s, but the global limit is much stricter and affects all resource types simultaneously, causing multi-minute lockouts at scale)").Short('p').Default("5m").Duration()
		maxReconcileRate         = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()
		syncPeriod               = app.Flag("sync", "How often all resources will be double-checked for drift from the desired state.").Short('s').Default("1h").Duration()
		enableManagementPolicies = app.Flag("enable-management-policies", "Enable support for management policies.").Default("true").OverrideDefaultFromEnvar("ENABLE_MANAGEMENT_POLICIES").Bool()
		pollStateMetricInterval  = app.Flag("poll-state-metric", "State metric recording interval").Default("5s").Duration()
		metricsBindAddress       = app.Flag("metrics-bind-address", "The address the metrics endpoint binds to.").Default(":8080").String()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	var zl = sigzap.New(sigzap.UseDevMode(*debug), func(o *sigzap.Options) {
		if *debug {
			o.Level = zapcore.DebugLevel
		} else {
			o.Level = zapcore.InfoLevel
		}
	})

	log := logging.NewLogrLogger(zl.WithName("provider-discord"))

	shutdownTracing := tracing.Init("provider-discord")
	defer shutdownTracing(context.Background())

	// Always set the controller-runtime logger to capture reconciliation events
	// Use info level to avoid excessive verbosity while still showing important operations
	ctrl.SetLogger(zl.WithName("controller-runtime"))

	log.Info("Provider starting up",
		"provider", "provider-discord",
		"version", version.Version,
		"go-version", runtime.Version(),
		"platform", runtime.GOOS+"/"+runtime.GOARCH,
		"sync-period", syncPeriod.String(),
		"poll-interval", pollInterval.String(),
		"max-reconcile-rate", *maxReconcileRate,
		"leader-election", *leaderElection,
		"leader-election-namespace", *leaderElectionNS,
		"management-policies", *enableManagementPolicies,
		"debug-mode", *debug)

	s := apimachineryruntime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		kingpin.FatalIfError(err, "Cannot add k8s types to scheme")
	}
	if err := apis.AddToScheme(s); err != nil {
		kingpin.FatalIfError(err, "Cannot add Discord APIs to scheme")
	}

	cfg, err := ctrl.GetConfig()
	if err != nil {
		kingpin.FatalIfError(err, "Cannot get API server rest config")
	}

	mgr, err := ctrl.NewManager(ratelimiter.LimitRESTConfig(cfg, *maxReconcileRate), ctrl.Options{
		Cache: cache.Options{
			SyncPeriod: syncPeriod,
		},
		LeaderElection:             *leaderElection,
		LeaderElectionID:           "crossplane-leader-election-provider-discord",
		LeaderElectionNamespace:    *leaderElectionNS,
		LeaderElectionResourceLock: "leases",
		Scheme:                     s,
		Metrics: metricserver.Options{
			BindAddress: *metricsBindAddress,
		},
		LeaseDuration: func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline: func() *time.Duration { d := 50 * time.Second; return &d }(),
		Controller: config.Controller{
			CacheSyncTimeout: 10 * time.Minute,
		},
	})
	if err != nil {
		kingpin.FatalIfError(err, "Cannot create controller manager")
	}

	mrStateMetrics := statemetrics.NewMRStateMetrics()
	crmetrics.Registry.MustRegister(mrStateMetrics)

	mo := xpcontroller.MetricOptions{
		PollStateMetricInterval: *pollStateMetricInterval,
		MRStateMetrics:          mrStateMetrics,
	}

	o := xpcontroller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
		Features:                &feature.Flags{},
		MetricOptions:           &mo,
	}

	if *enableManagementPolicies {
		o.Features.Enable(features.EnableAlphaManagementPolicies)
		log.Info("Alpha feature enabled", "flag", features.EnableAlphaManagementPolicies)
	}

	// Initialize metrics recorder for Discord API monitoring
	metricsRecorder := metrics.NewMetricsRecorder()

	log.Info("Setting up Discord controllers")
	if err := controller.SetupWithMetrics(mgr, o, metricsRecorder); err != nil {
		kingpin.FatalIfError(err, "Cannot setup Discord controllers")
	}
	log.Info("Successfully set up Discord controllers")

	// Register state metrics for managed resources
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &guildv1beta1.GuildList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Guild")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &channelv1beta1.ChannelList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Channel")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &rolev1beta1.RoleList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Role")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &memberv1beta1.MemberList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Member")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &invitev1beta1.InviteList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Invite")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &applicationv1beta1.ApplicationList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Application")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &integrationv1beta1.IntegrationList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Integration")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &webhookv1beta1.WebhookList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Webhook")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &userv1beta1.UserList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for User")

	kingpin.FatalIfError(mgr.AddHealthzCheck("healthz", healthz.Ping), "Cannot add health check")
	kingpin.FatalIfError(mgr.AddReadyzCheck("readyz", healthz.Ping), "Cannot add ready check")

	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
