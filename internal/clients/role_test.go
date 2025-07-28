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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRole(t *testing.T) {
	guildID := "123456789"
	expectedRole := Role{
		ID:          "987654321",
		Name:        "Test Role",
		Color:       16711680,
		Hoist:       true,
		Position:    1,
		Permissions: "1234567890",
		Managed:     false,
		Mentionable: true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/guilds/"+guildID+"/roles", r.URL.Path)
		assert.Equal(t, "Bot test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var req CreateRoleRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Test Role", req.Name)
		assert.Equal(t, 16711680, *req.Color)
		assert.Equal(t, true, *req.Hoist)
		assert.Equal(t, true, *req.Mentionable)
		assert.Equal(t, "1234567890", *req.Permissions)

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedRole)
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	color := 16711680
	hoist := true
	mentionable := true
	permissions := "1234567890"

	req := CreateRoleRequest{
		Name:        "Test Role",
		Color:       &color,
		Hoist:       &hoist,
		Mentionable: &mentionable,
		Permissions: &permissions,
	}

	role, err := client.CreateRole(context.Background(), guildID, req)
	require.NoError(t, err)
	assert.Equal(t, expectedRole.ID, role.ID)
	assert.Equal(t, expectedRole.Name, role.Name)
	assert.Equal(t, expectedRole.Color, role.Color)
	assert.Equal(t, expectedRole.Hoist, role.Hoist)
	assert.Equal(t, expectedRole.Mentionable, role.Mentionable)
	assert.Equal(t, expectedRole.Permissions, role.Permissions)
}

func TestGetRole(t *testing.T) {
	guildID := "123456789"
	roleID := "987654321"
	
	roles := []Role{
		{
			ID:          "987654321",
			Name:        "Test Role",
			Color:       16711680,
			Hoist:       true,
			Position:    1,
			Permissions: "1234567890",
			Managed:     false,
			Mentionable: true,
		},
		{
			ID:          "111111111",
			Name:        "Other Role",
			Color:       0,
			Hoist:       false,
			Position:    0,
			Permissions: "0",
			Managed:     false,
			Mentionable: false,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/guilds/"+guildID+"/roles", r.URL.Path)
		assert.Equal(t, "Bot test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(roles)
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	role, err := client.GetRole(context.Background(), guildID, roleID)
	require.NoError(t, err)
	assert.Equal(t, roles[0].ID, role.ID)
	assert.Equal(t, roles[0].Name, role.Name)
	assert.Equal(t, roles[0].Color, role.Color)
}

func TestGetRoleNotFound(t *testing.T) {
	guildID := "123456789"
	roleID := "nonexistent"
	
	roles := []Role{
		{
			ID:   "987654321",
			Name: "Test Role",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(roles)
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	role, err := client.GetRole(context.Background(), guildID, roleID)
	assert.Error(t, err)
	assert.Nil(t, role)
	assert.Contains(t, err.Error(), "role not found")
}

func TestModifyRole(t *testing.T) {
	guildID := "123456789"
	roleID := "987654321"
	
	expectedRole := Role{
		ID:          roleID,
		Name:        "Modified Role",
		Color:       255,
		Hoist:       false,
		Position:    2,
		Permissions: "9876543210",
		Managed:     false,
		Mentionable: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/guilds/"+guildID+"/roles/"+roleID, r.URL.Path)
		assert.Equal(t, "Bot test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var req ModifyRoleRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Modified Role", *req.Name)
		assert.Equal(t, 255, *req.Color)
		assert.Equal(t, false, *req.Hoist)
		assert.Equal(t, 2, *req.Position)
		assert.Equal(t, false, *req.Mentionable)
		assert.Equal(t, "9876543210", *req.Permissions)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedRole)
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	name := "Modified Role"
	color := 255
	hoist := false
	position := 2
	mentionable := false
	permissions := "9876543210"

	req := ModifyRoleRequest{
		Name:        &name,
		Color:       &color,
		Hoist:       &hoist,
		Position:    &position,
		Mentionable: &mentionable,
		Permissions: &permissions,
	}

	role, err := client.ModifyRole(context.Background(), guildID, roleID, req)
	require.NoError(t, err)
	assert.Equal(t, expectedRole.ID, role.ID)
	assert.Equal(t, expectedRole.Name, role.Name)
	assert.Equal(t, expectedRole.Color, role.Color)
	assert.Equal(t, expectedRole.Hoist, role.Hoist)
	assert.Equal(t, expectedRole.Position, role.Position)
	assert.Equal(t, expectedRole.Mentionable, role.Mentionable)
	assert.Equal(t, expectedRole.Permissions, role.Permissions)
}

func TestDeleteRole(t *testing.T) {
	guildID := "123456789"
	roleID := "987654321"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/guilds/"+guildID+"/roles/"+roleID, r.URL.Path)
		assert.Equal(t, "Bot test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteRole(context.Background(), guildID, roleID)
	assert.NoError(t, err)
}

func TestRoleErrorHandling(t *testing.T) {
	guildID := "123456789"
	roleID := "987654321"

	tests := []struct {
		name       string
		statusCode int
		method     string
		endpoint   string
		operation  func(*DiscordClient) error
	}{
		{
			name:       "CreateRole 400 error",
			statusCode: http.StatusBadRequest,
			method:     "POST",
			endpoint:   "/guilds/" + guildID + "/roles",
			operation: func(c *DiscordClient) error {
				_, err := c.CreateRole(context.Background(), guildID, CreateRoleRequest{Name: "Test"})
				return err
			},
		},
		{
			name:       "GetRole 404 error",
			statusCode: http.StatusNotFound,
			method:     "GET", 
			endpoint:   "/guilds/" + guildID + "/roles",
			operation: func(c *DiscordClient) error {
				_, err := c.GetRole(context.Background(), guildID, roleID)
				return err
			},
		},
		{
			name:       "ModifyRole 403 error",
			statusCode: http.StatusForbidden,
			method:     "PATCH",
			endpoint:   "/guilds/" + guildID + "/roles/" + roleID,
			operation: func(c *DiscordClient) error {
				name := "Modified"
				_, err := c.ModifyRole(context.Background(), guildID, roleID, ModifyRoleRequest{Name: &name})
				return err
			},
		},
		{
			name:       "DeleteRole 401 error",
			statusCode: http.StatusUnauthorized,
			method:     "DELETE",
			endpoint:   "/guilds/" + guildID + "/roles/" + roleID,
			operation: func(c *DiscordClient) error {
				return c.DeleteRole(context.Background(), guildID, roleID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.method, r.Method)
				assert.True(t, strings.HasPrefix(r.URL.Path, tt.endpoint) || r.URL.Path == tt.endpoint)
				
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"message": "Error occurred", "code": 50013}`))
			}))
			defer server.Close()

			client := NewDiscordClient("test-token")
			client.baseURL = server.URL

			err := tt.operation(client)
			assert.Error(t, err)
		})
	}
}