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

package clients

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/rossigee/provider-discord/apis/v1beta1"
)

const (
	errNoProviderConfig     = "no providerConfigRef provided"
	errGetProviderConfig    = "cannot get referenced ProviderConfig"
	errTrackUsage          = "cannot track ProviderConfig usage"
	errExtractCredentials   = "cannot extract credentials"
	errUnmarshalCredentials = "cannot unmarshal credentials"
)

// CredentialsSource is the source of the credentials for Discord API
type CredentialsSource string

const (
	// CredentialsSourceSecret indicates credentials should be fetched from a secret
	CredentialsSourceSecret CredentialsSource = "Secret"
)

// ProviderCredentials holds the configuration for Discord API credentials
type ProviderCredentials struct {
	// Source is the source of the credentials
	Source CredentialsSource `json:"source"`

	// BotToken is the Discord bot token
	BotToken string `json:"botToken,omitempty"`

	xpv1.CommonCredentialSelectors `json:",inline"`
}

// Extract extracts Discord credentials from the referenced secret
func (c *ProviderCredentials) Extract(ctx context.Context, client client.Client) (string, error) {
	if c.Source != CredentialsSourceSecret {
		return "", errors.New("only secret source is supported")
	}

	if c.SecretRef == nil {
		return "", errors.New("no secret reference provided")
	}

	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{
		Namespace: c.SecretRef.Namespace,
		Name:      c.SecretRef.Name,
	}, secret); err != nil {
		return "", errors.Wrap(err, "cannot get credentials secret")
	}

	token, ok := secret.Data[c.SecretRef.Key]
	if !ok {
		return "", errors.Errorf("credentials secret does not contain key %s", c.SecretRef.Key)
	}

	return string(token), nil
}

// GetConfig extracts the Discord bot token from a ProviderConfig
func GetConfig(ctx context.Context, c client.Client, mg resource.Managed) (*string, error) {
	pc := &v1beta1.ProviderConfig{}
	if err := c.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	t := resource.NewProviderConfigUsageTracker(c, &v1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}

	// Extract token from the credentials
	if pc.Spec.Credentials.Source != xpv1.CredentialsSourceSecret {
		return nil, errors.New("only secret source is supported")
	}

	if pc.Spec.Credentials.SecretRef == nil {
		return nil, errors.New("no secret reference provided")
	}

	secret := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{
		Namespace: pc.Spec.Credentials.SecretRef.Namespace,
		Name:      pc.Spec.Credentials.SecretRef.Name,
	}, secret); err != nil {
		return nil, errors.Wrap(err, "cannot get credentials secret")
	}

	tokenBytes, ok := secret.Data[pc.Spec.Credentials.SecretRef.Key]
	if !ok {
		return nil, errors.Errorf("credentials secret does not contain key %s", pc.Spec.Credentials.SecretRef.Key)
	}

	token := string(tokenBytes)
	return &token, nil
}