package webhook

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	webhookv1beta1 "github.com/rossigee/provider-discord/apis/webhook/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// An observed webhook must report Ready=True. Before the fix, Observe never
// set the Ready condition, so a webhook left in Creating by an earlier Create
// stayed Creating forever even though Discord had the webhook.
func TestObserveExistingWebhookSetsReady(t *testing.T) {
	const webhookID = "1573621321345278023"
	const channelID = "1573488275224526878"

	cr := &webhookv1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{meta.AnnotationKeyExternalName: webhookID},
		},
		Spec: webhookv1beta1.WebhookSpec{
			ForProvider: webhookv1beta1.WebhookParameters{
				Name:      "alerts",
				ChannelID: channelID,
			},
		},
	}
	cr.SetConditions(xpv1.Creating())

	mockClient := &MockWebhookClient{
		GetWebhookFunc: func(_ context.Context, id string) (*discordclient.Webhook, error) {
			return &discordclient.Webhook{ID: id, Name: "alerts", ChannelID: channelID}, nil
		},
	}
	e := &external{service: mockClient, kube: nil}

	obs, err := e.Observe(context.Background(), cr)
	assert.NoError(t, err)
	assert.True(t, obs.ResourceExists)
	assert.Equal(t, xpv1.Available().Status, cr.GetCondition(xpv1.TypeReady).Status)
	assert.Equal(t, xpv1.TypeReady, cr.GetCondition(xpv1.TypeReady).Type)
	assert.Equal(t, "Available", string(cr.GetCondition(xpv1.TypeReady).Reason))
}

// Update succeeding must also leave the webhook Ready.
func TestUpdateSetsReady(t *testing.T) {
	const webhookID = "1573621321345278023"

	cr := &webhookv1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{meta.AnnotationKeyExternalName: webhookID},
		},
		Spec: webhookv1beta1.WebhookSpec{
			ForProvider: webhookv1beta1.WebhookParameters{Name: "alerts"},
		},
	}
	cr.SetConditions(xpv1.Creating())

	mockClient := &MockWebhookClient{
		ModifyWebhookFunc: func(_ context.Context, id string, _ *discordclient.ModifyWebhookRequest) (*discordclient.Webhook, error) {
			return &discordclient.Webhook{ID: id, Name: "alerts"}, nil
		},
	}
	e := &external{service: mockClient, kube: nil}

	_, err := e.Update(context.Background(), cr)
	assert.NoError(t, err)
	assert.Equal(t, "Available", string(cr.GetCondition(xpv1.TypeReady).Reason))
}
