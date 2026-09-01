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

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDisconnect validates the disconnect method
func TestDisconnect(t *testing.T) {
	e := &external{}
	err := e.Disconnect(context.Background())
	assert.NoError(t, err)
}

// TestObserveTypeAssertion validates type checking in Observe
func TestObserveTypeAssertion(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Observe(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an Integration")
}

// TestCreateTypeAssertion validates type checking in Create
func TestCreateTypeAssertion(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Create(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an Integration")
}

// TestUpdateTypeAssertion validates type checking in Update
func TestUpdateTypeAssertion(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Update(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an Integration")
}

// TestDeleteTypeAssertion validates type checking in Delete
func TestDeleteTypeAssertion(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Delete(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an Integration")
}
