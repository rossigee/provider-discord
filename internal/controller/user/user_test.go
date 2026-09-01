package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisconnect(t *testing.T) {
	e := &external{}
	err := e.Disconnect(context.Background())
	assert.NoError(t, err)
}

func TestObserveTypeAssertion(t *testing.T) {
	_, err := (&external{}).Observe(context.Background(), nil)
	assert.Error(t, err)
}

func TestCreateTypeAssertion(t *testing.T) {
	_, err := (&external{}).Create(context.Background(), nil)
	assert.Error(t, err)
}

func TestUpdateTypeAssertion(t *testing.T) {
	_, err := (&external{}).Update(context.Background(), nil)
	assert.Error(t, err)
}

func TestDeleteTypeAssertion(t *testing.T) {
	_, err := (&external{}).Delete(context.Background(), nil)
	assert.Error(t, err)
}
