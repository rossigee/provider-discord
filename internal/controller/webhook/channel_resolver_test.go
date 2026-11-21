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

package webhook

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	channelv1beta1 "github.com/rossigee/provider-discord/apis/channel/v1beta1"
	webhookv1beta1 "github.com/rossigee/provider-discord/apis/webhook/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace  = "bankrut"
	testChannelID  = "1573488275224526878"
	testChannelRef = "golder-chat-alerts"
)

func newTestChannel(ready corev1.ConditionStatus, externalName string) *channelv1beta1.Channel {
	ch := &channelv1beta1.Channel{
		ObjectMeta: metav1.ObjectMeta{Name: testChannelRef, Namespace: testNamespace},
	}
	if externalName != "" {
		meta.SetExternalName(ch, externalName)
	}
	ch.SetConditions(xpv1.Condition{Type: xpv1.TypeReady, Status: ready})
	return ch
}

func newTestWebhook(ref *xpv1.Reference) *webhookv1beta1.Webhook {
	return &webhookv1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{Name: "ban-krut-notifications-iot", Namespace: testNamespace},
		Spec: webhookv1beta1.WebhookSpec{
			ForProvider: webhookv1beta1.WebhookParameters{
				Name:       "bankrut-notifications iot",
				ChannelRef: ref,
			},
		},
	}
}

func newTestResolver(t *testing.T, objs ...client.Object) *channelReferenceResolver {
	t.Helper()
	s := runtime.NewScheme()
	if err := channelv1beta1.AddToScheme(s); err != nil {
		t.Fatalf("add channel scheme: %v", err)
	}
	return &channelReferenceResolver{kube: fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()}
}

func TestResolveReferencesNoRef(t *testing.T) {
	r := newTestResolver(t)
	cr := newTestWebhook(nil)
	cr.Spec.ForProvider.ChannelID = "1559545778450599976"

	if err := r.ResolveReferences(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cr.Spec.ForProvider.ChannelID != "1559545778450599976" {
		t.Errorf("channelId changed without a channelRef: %q", cr.Spec.ForProvider.ChannelID)
	}
}

func TestResolveReferencesSetsCurrentChannelID(t *testing.T) {
	r := newTestResolver(t, newTestChannel(corev1.ConditionTrue, testChannelID))
	cr := newTestWebhook(&xpv1.Reference{Name: testChannelRef})
	cr.Spec.ForProvider.ChannelID = "1559545778450599976" // stale value, must be overwritten

	if err := r.ResolveReferences(context.Background(), cr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cr.Spec.ForProvider.ChannelID != testChannelID {
		t.Errorf("channelId = %q, want %q", cr.Spec.ForProvider.ChannelID, testChannelID)
	}
}

func TestResolveReferencesChannelMissing(t *testing.T) {
	r := newTestResolver(t)
	cr := newTestWebhook(&xpv1.Reference{Name: testChannelRef})

	if err := r.ResolveReferences(context.Background(), cr); err == nil {
		t.Fatal("expected error for missing channel, got nil")
	}
	if cr.Spec.ForProvider.ChannelID != "" {
		t.Errorf("channelId set despite missing channel: %q", cr.Spec.ForProvider.ChannelID)
	}
}

func TestResolveReferencesChannelNotReady(t *testing.T) {
	r := newTestResolver(t, newTestChannel(corev1.ConditionFalse, testChannelID))
	cr := newTestWebhook(&xpv1.Reference{Name: testChannelRef})

	if err := r.ResolveReferences(context.Background(), cr); err == nil {
		t.Fatal("expected error for channel that is not Ready, got nil")
	}
	if cr.Spec.ForProvider.ChannelID != "" {
		t.Errorf("channelId set from a non-Ready channel: %q", cr.Spec.ForProvider.ChannelID)
	}
}

func TestResolveReferencesNoDiscordID(t *testing.T) {
	r := newTestResolver(t, newTestChannel(corev1.ConditionTrue, ""))
	cr := newTestWebhook(&xpv1.Reference{Name: testChannelRef})

	if err := r.ResolveReferences(context.Background(), cr); err == nil {
		t.Fatal("expected error for channel without a Discord ID, got nil")
	}
	if cr.Spec.ForProvider.ChannelID != "" {
		t.Errorf("channelId set without a Discord ID: %q", cr.Spec.ForProvider.ChannelID)
	}
}
