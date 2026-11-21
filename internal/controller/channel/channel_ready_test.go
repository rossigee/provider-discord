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

package channel

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	channelv1beta1 "github.com/rossigee/provider-discord/apis/channel/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// An observed channel must report Ready=True. Before the fix, Observe never set
// the Ready condition, so a channel left in Creating by an earlier Create stayed
// Creating forever even though Discord had the channel.
func TestObserveExistingChannelSetsReady(t *testing.T) {
	const guildID = "123456789012345678"
	const channelID = "987654321098765432"

	cr := &channelv1beta1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{meta.AnnotationKeyExternalName: channelID},
		},
		Spec: channelv1beta1.ChannelSpec{
			ForProvider: channelv1beta1.ChannelParameters{Name: "alerts", Type: 0, GuildID: guildID},
		},
	}
	cr.SetConditions(xpv1.Creating())

	mockClient := &MockChannelClient{
		GetChannelFunc: func(ctx context.Context, id string) (*discordclient.Channel, error) {
			return &discordclient.Channel{ID: id, Name: "alerts", Type: 0, GuildID: guildID}, nil
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

// Update succeeding must also leave the channel Ready.
func TestUpdateSetsReady(t *testing.T) {
	const guildID = "123456789012345678"
	const channelID = "987654321098765432"

	cr := &channelv1beta1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{meta.AnnotationKeyExternalName: channelID},
		},
		Spec: channelv1beta1.ChannelSpec{
			ForProvider: channelv1beta1.ChannelParameters{Name: "alerts", Type: 0, GuildID: guildID},
		},
	}
	cr.SetConditions(xpv1.Creating())

	mockClient := &MockChannelClient{
		ModifyChannelFunc: func(ctx context.Context, id string, req *discordclient.ModifyChannelRequest) (*discordclient.Channel, error) {
			return &discordclient.Channel{ID: id, Name: "alerts", Type: 0, GuildID: guildID}, nil
		},
	}
	e := &external{service: mockClient, kube: nil}

	_, err := e.Update(context.Background(), cr)
	assert.NoError(t, err)
	assert.Equal(t, "Available", string(cr.GetCondition(xpv1.TypeReady).Reason))
}
