//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:allowDangerousTypes=true paths="./..." output:crd:artifacts:config=../../package/crds object:headerFile=../../hack/boilerplate.go.txt
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../hack/boilerplate.go.txt paths="./..."

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

package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "discord.crossplane.io"
	Version = "v1alpha1"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// ProviderConfig type metadata.
var (
	ProviderConfigKind             = reflect.TypeOf(ProviderConfig{}).Name()
	ProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ProviderConfigKind}.String()
	ProviderConfigKindAPIVersion   = ProviderConfigKind + "." + GroupVersion.String()
	ProviderConfigGroupVersionKind = GroupVersion.WithKind(ProviderConfigKind)

	ProviderConfigUsageKind             = reflect.TypeOf(ProviderConfigUsage{}).Name()
	ProviderConfigUsageGroupKind        = schema.GroupKind{Group: Group, Kind: ProviderConfigUsageKind}.String()
	ProviderConfigUsageKindAPIVersion   = ProviderConfigUsageKind + "." + GroupVersion.String()
	ProviderConfigUsageGroupVersionKind = GroupVersion.WithKind(ProviderConfigUsageKind)

	ProviderConfigUsageListKind             = reflect.TypeOf(ProviderConfigUsageList{}).Name()
	ProviderConfigUsageListGroupKind        = schema.GroupKind{Group: Group, Kind: ProviderConfigUsageListKind}.String()
	ProviderConfigUsageListKindAPIVersion   = ProviderConfigUsageListKind + "." + GroupVersion.String()
	ProviderConfigUsageListGroupVersionKind = GroupVersion.WithKind(ProviderConfigUsageListKind)
)

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
	SchemeBuilder.Register(&ProviderConfigUsage{}, &ProviderConfigUsageList{})
}