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

// Package apis contains Kubernetes API for the Discord provider.
package apis

import (
	applicationv1beta1 "github.com/rossigee/provider-discord/apis/application/v1beta1"
	channelv1beta1 "github.com/rossigee/provider-discord/apis/channel/v1beta1"
	deduplicationv1beta1 "github.com/rossigee/provider-discord/apis/deduplication/v1beta1"
	guildv1beta1 "github.com/rossigee/provider-discord/apis/guild/v1beta1"
	integrationv1beta1 "github.com/rossigee/provider-discord/apis/integration/v1beta1"
	invitev1beta1 "github.com/rossigee/provider-discord/apis/invite/v1beta1"
	memberv1beta1 "github.com/rossigee/provider-discord/apis/member/v1beta1"
	rolev1beta1 "github.com/rossigee/provider-discord/apis/role/v1beta1"
	userv1beta1 "github.com/rossigee/provider-discord/apis/user/v1beta1"
	"github.com/rossigee/provider-discord/apis/v1beta1"
	webhookv1beta1 "github.com/rossigee/provider-discord/apis/webhook/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	AddToSchemes = append(AddToSchemes,
		v1beta1.AddToScheme,
		guildv1beta1.AddToScheme,
		channelv1beta1.AddToScheme,
		rolev1beta1.AddToScheme,
		webhookv1beta1.AddToScheme,
		invitev1beta1.AddToScheme,
		memberv1beta1.AddToScheme,
		userv1beta1.AddToScheme,
		applicationv1beta1.AddToScheme,
		integrationv1beta1.AddToScheme,
		deduplicationv1beta1.AddToScheme,
	)
}

var AddToSchemes runtime.SchemeBuilder

func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
