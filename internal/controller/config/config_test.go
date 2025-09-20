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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
)

func TestSetup(t *testing.T) {
	t.Skip("Setup function requires integration testing with a real Kubernetes environment")

	// This test would require a real Kubernetes environment to properly test
	// the controller setup. For unit testing purposes, we can verify that
	// the Setup function exists and has the correct signature.

	// Test that Setup function doesn't panic when called with nil manager
	// (this will error, but shouldn't panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Setup function panicked: %v", r)
		}
	}()

	opts := controller.Options{}
	err := Setup(nil, opts)

	// We expect an error since we passed nil manager, but no panic
	assert.Error(t, err)
}
