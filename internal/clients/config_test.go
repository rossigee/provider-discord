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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-discord/apis/v1beta1"
)

var (
	testNamespace = "test-namespace"
	testName      = "test-name"
	testKey       = "token"
	testToken     = "discord-bot-token-123"
)

func TestGetConfig(t *testing.T) {
	type args struct {
		mg      resource.Managed
		objects []client.Object
	}
	type want struct {
		token string
		err   error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"Success": {
			reason: "Should extract token from secret successfully",
			args: args{
				mg: &MockManaged{
					providerConfigRef: &xpv1.Reference{
						Name: "test-provider-config",
					},
				},
				objects: []client.Object{
					&v1beta1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-provider-config",
						},
						Spec: v1beta1.ProviderConfigSpec{
							Credentials: v1beta1.ProviderCredentials{
								Source: xpv1.CredentialsSourceSecret,
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      testName,
											Namespace: testNamespace,
										},
										Key: testKey,
									},
								},
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testName,
							Namespace: testNamespace,
						},
						Data: map[string][]byte{
							testKey: []byte(testToken),
						},
					},
				},
			},
			want: want{
				token: testToken,
			},
		},
		"ProviderConfigNotFound": {
			reason: "Should fail when provider config not found",
			args: args{
				mg: &MockManaged{
					providerConfigRef: &xpv1.Reference{
						Name: "non-existent",
					},
				},
				objects: []client.Object{},
			},
			want: want{
				err: errors.Wrap(errors.New("providerconfigs.discord.crossplane.io \"non-existent\" not found"), errGetProviderConfig),
			},
		},
		"SecretNotFound": {
			reason: "Should fail when secret not found",
			args: args{
				mg: &MockManaged{
					providerConfigRef: &xpv1.Reference{
						Name: "test-provider-config",
					},
				},
				objects: []client.Object{
					&v1beta1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-provider-config",
						},
						Spec: v1beta1.ProviderConfigSpec{
							Credentials: v1beta1.ProviderCredentials{
								Source: xpv1.CredentialsSourceSecret,
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "non-existent-secret",
											Namespace: testNamespace,
										},
										Key: testKey,
									},
								},
							},
						},
					},
				},
			},
			want: want{
				err: errors.New("cannot get credentials secret"),
			},
		},
		"KeyNotInSecret": {
			reason: "Should fail when key not found in secret",
			args: args{
				mg: &MockManaged{
					providerConfigRef: &xpv1.Reference{
						Name: "test-provider-config",
					},
				},
				objects: []client.Object{
					&v1beta1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-provider-config",
						},
						Spec: v1beta1.ProviderConfigSpec{
							Credentials: v1beta1.ProviderCredentials{
								Source: xpv1.CredentialsSourceSecret,
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      testName,
											Namespace: testNamespace,
										},
										Key: "wrong-key",
									},
								},
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testName,
							Namespace: testNamespace,
						},
						Data: map[string][]byte{
							testKey: []byte(testToken),
						},
					},
				},
			},
			want: want{
				err: errors.Errorf("credentials secret does not contain key wrong-key"),
			},
		},
		"NoSecretRef": {
			reason: "Should fail when no secret reference provided",
			args: args{
				mg: &MockManaged{
					providerConfigRef: &xpv1.Reference{
						Name: "test-provider-config",
					},
				},
				objects: []client.Object{
					&v1beta1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-provider-config",
						},
						Spec: v1beta1.ProviderConfigSpec{
							Credentials: v1beta1.ProviderCredentials{
								Source: xpv1.CredentialsSourceSecret,
							},
						},
					},
				},
			},
			want: want{
				err: errors.New("no secret reference provided"),
			},
		},
	}

	// Create scheme once
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Create fake client with scheme and objects
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.args.objects...).
				Build()

			token, err := GetConfig(context.Background(), fakeClient, tc.args.mg)

			// Check error
			if tc.want.err != nil {
				if err == nil {
					t.Errorf("%s: expected error %v, got nil", tc.reason, tc.want.err)
					return
				}
				// For "not found" errors, just check the message contains expected text
				if !contains(err.Error(), tc.want.err.Error()) {
					t.Errorf("%s: expected error containing %q, got %q", tc.reason, tc.want.err.Error(), err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("%s: unexpected error: %v", tc.reason, err)
				return
			}

			// Check token
			if token == nil {
				t.Errorf("%s: expected token, got nil", tc.reason)
				return
			}
			if diff := cmp.Diff(tc.want.token, *token); diff != "" {
				t.Errorf("%s: -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// MockManaged is a mock implementation of resource.Managed
type MockManaged struct {
	providerConfigRef *xpv1.Reference
}

func (m *MockManaged) GetProviderConfigReference() *xpv1.Reference {
	return m.providerConfigRef
}

func (m *MockManaged) SetProviderConfigReference(r *xpv1.Reference) {}

func (m *MockManaged) GetProviderReference() *xpv1.Reference {
	return nil
}

func (m *MockManaged) SetProviderReference(r *xpv1.Reference) {}

func (m *MockManaged) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return nil
}

func (m *MockManaged) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {}

func (m *MockManaged) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return xpv1.Condition{}
}

func (m *MockManaged) SetConditions(c ...xpv1.Condition) {}

func (m *MockManaged) GetDeletionPolicy() xpv1.DeletionPolicy {
	return ""
}

func (m *MockManaged) SetDeletionPolicy(p xpv1.DeletionPolicy) {}

func (m *MockManaged) GetManagementPolicies() xpv1.ManagementPolicies {
	return nil
}

func (m *MockManaged) SetManagementPolicies(p xpv1.ManagementPolicies) {}

func (m *MockManaged) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (m *MockManaged) DeepCopyObject() runtime.Object {
	return m
}

func (m *MockManaged) GetUID() types.UID {
	return "test-uid"
}

func (m *MockManaged) GetName() string {
	return "test-managed"
}

func (m *MockManaged) GetNamespace() string {
	return "test-namespace"
}

func (m *MockManaged) GetLabels() map[string]string {
	return map[string]string{}
}

func (m *MockManaged) SetLabels(labels map[string]string) {}

func (m *MockManaged) GetAnnotations() map[string]string {
	return map[string]string{}
}

func (m *MockManaged) SetAnnotations(annotations map[string]string) {}

func (m *MockManaged) GetFinalizers() []string {
	return []string{}
}

func (m *MockManaged) SetFinalizers(finalizers []string) {}

func (m *MockManaged) GetOwnerReferences() []metav1.OwnerReference {
	return []metav1.OwnerReference{}
}

func (m *MockManaged) SetOwnerReferences([]metav1.OwnerReference) {}

func (m *MockManaged) GetGenerateName() string {
	return ""
}

func (m *MockManaged) SetGenerateName(name string) {}

func (m *MockManaged) SetUID(uid types.UID) {}

func (m *MockManaged) SetName(name string) {}

func (m *MockManaged) SetNamespace(namespace string) {}

func (m *MockManaged) GetResourceVersion() string {
	return ""
}

func (m *MockManaged) SetResourceVersion(version string) {}

func (m *MockManaged) GetGeneration() int64 {
	return 0
}

func (m *MockManaged) SetGeneration(generation int64) {}

func (m *MockManaged) GetSelfLink() string {
	return ""
}

func (m *MockManaged) SetSelfLink(selfLink string) {}

func (m *MockManaged) GetCreationTimestamp() metav1.Time {
	return metav1.Time{}
}

func (m *MockManaged) SetCreationTimestamp(timestamp metav1.Time) {}

func (m *MockManaged) GetDeletionTimestamp() *metav1.Time {
	return nil
}

func (m *MockManaged) SetDeletionTimestamp(timestamp *metav1.Time) {}

func (m *MockManaged) GetDeletionGracePeriodSeconds() *int64 {
	return nil
}

func (m *MockManaged) SetDeletionGracePeriodSeconds(*int64) {}

func (m *MockManaged) GetManagedFields() []metav1.ManagedFieldsEntry {
	return nil
}

func (m *MockManaged) SetManagedFields(managedFields []metav1.ManagedFieldsEntry) {}

func TestExtract(t *testing.T) {
	type args struct {
		credentials ProviderCredentials
		objects     []client.Object
	}
	type want struct {
		token string
		err   error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"Success": {
			reason: "Should extract token from secret successfully",
			args: args{
				credentials: ProviderCredentials{
					Source: CredentialsSourceSecret,
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      testName,
								Namespace: testNamespace,
							},
							Key: testKey,
						},
					},
				},
				objects: []client.Object{
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testName,
							Namespace: testNamespace,
						},
						Data: map[string][]byte{
							testKey: []byte(testToken),
						},
					},
				},
			},
			want: want{
				token: testToken,
			},
		},
		"UnsupportedSource": {
			reason: "Should fail with unsupported source",
			args: args{
				credentials: ProviderCredentials{
					Source: "environment",
				},
			},
			want: want{
				err: errors.New("only secret source is supported"),
			},
		},
		"NoSecretRef": {
			reason: "Should fail when no secret reference provided",
			args: args{
				credentials: ProviderCredentials{
					Source: CredentialsSourceSecret,
				},
			},
			want: want{
				err: errors.New("no secret reference provided"),
			},
		},
		"SecretNotFound": {
			reason: "Should fail when secret not found",
			args: args{
				credentials: ProviderCredentials{
					Source: CredentialsSourceSecret,
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      "non-existent",
								Namespace: testNamespace,
							},
							Key: testKey,
						},
					},
				},
				objects: []client.Object{},
			},
			want: want{
				err: errors.New("cannot get credentials secret"),
			},
		},
		"KeyNotInSecret": {
			reason: "Should fail when key not found in secret",
			args: args{
				credentials: ProviderCredentials{
					Source: CredentialsSourceSecret,
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      testName,
								Namespace: testNamespace,
							},
							Key: "wrong-key",
						},
					},
				},
				objects: []client.Object{
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testName,
							Namespace: testNamespace,
						},
						Data: map[string][]byte{
							testKey: []byte(testToken),
						},
					},
				},
			},
			want: want{
				err: errors.Errorf("credentials secret does not contain key wrong-key"),
			},
		},
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.args.objects...).
				Build()

			token, err := tc.args.credentials.Extract(context.Background(), fakeClient)

			if tc.want.err != nil {
				if err == nil {
					t.Errorf("%s: expected error %v, got nil", tc.reason, tc.want.err)
					return
				}
				if !contains(err.Error(), tc.want.err.Error()) {
					t.Errorf("%s: expected error containing %q, got %q", tc.reason, tc.want.err.Error(), err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("%s: unexpected error: %v", tc.reason, err)
				return
			}

			if diff := cmp.Diff(tc.want.token, token); diff != "" {
				t.Errorf("%s: -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}