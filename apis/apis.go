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
	"k8s.io/apimachinery/pkg/runtime"

	applicationv1alpha1 "github.com/rossigee/provider-discord/apis/application/v1alpha1"
	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	guildv1alpha1 "github.com/rossigee/provider-discord/apis/guild/v1alpha1"
	integrationv1alpha1 "github.com/rossigee/provider-discord/apis/integration/v1alpha1"
	invitev1alpha1 "github.com/rossigee/provider-discord/apis/invite/v1alpha1"
	memberv1alpha1 "github.com/rossigee/provider-discord/apis/member/v1alpha1"
	rolev1alpha1 "github.com/rossigee/provider-discord/apis/role/v1alpha1"
	userv1alpha1 "github.com/rossigee/provider-discord/apis/user/v1alpha1"
	v1beta1 "github.com/rossigee/provider-discord/apis/v1beta1"
	webhookv1alpha1 "github.com/rossigee/provider-discord/apis/webhook/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1beta1.AddToScheme,
		guildv1alpha1.AddToScheme,
		channelv1alpha1.AddToScheme,
		rolev1alpha1.AddToScheme,
		webhookv1alpha1.AddToScheme,
		invitev1alpha1.AddToScheme,
		memberv1alpha1.AddToScheme,
		userv1alpha1.AddToScheme,
		applicationv1alpha1.AddToScheme,
		integrationv1alpha1.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
