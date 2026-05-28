/*
Copyright 2026.

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

package keys

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openkruise/agents/pkg/servers/e2b/models"
)


func TestGenerateAPIKey_HasCorrectPrefix(t *testing.T) {
	key, err := GenerateAPIKey()
	require.NoError(t, err)
	assert.Equal(t, APIKeyPrefix, key[:len(APIKeyPrefix)])
}

func TestIsValidAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		expect bool
	}{
		{
			name:   "valid 40-char key",
			key:    "e2b_0123456789abcdef0123456789abcdef01234567",
			expect: true,
		},
		{
			name:   "valid shorter key",
			key:    "e2b_abcdef0123456789",
			expect: true,
		},
		{
			name:   "missing prefix",
			key:    "sk_0123456789abcdef",
			expect: false,
		},
		{
			name:   "prefix only without body",
			key:    "e2b_",
			expect: false,
		},
		{
			name:   "empty string",
			key:    "",
			expect: false,
		},
		{
			name:   "uuid format without prefix",
			key:    "550e8400-e29b-41d4-a716-446655440000",
			expect: false,
		},
		{
			name:   "uppercase hex rejected",
			key:    "e2b_0123456789ABCDEF0123456789abcdef01234567",
			expect: false,
		},
		{
			name:   "non-hex characters rejected",
			key:    "e2b_zzzzzzzzzzzzzzzzzzzz",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, IsValidAPIKeyFormat(tt.key))
		})
	}
}
func TestTeamForKey(t *testing.T) {
	tests := []struct {
		name       string
		user       *models.CreatedTeamAPIKey
		expectTeam *models.Team
	}{
		{
			name:       "nil user defaults to admin team",
			user:       nil,
			expectTeam: models.AdminTeam(),
		},
		{
			name: "missing team defaults to admin team",
			user: &models.CreatedTeamAPIKey{
				ID: uuid.New(),
			},
			expectTeam: models.AdminTeam(),
		},
		{
			name: "admin team name is normalized to canonical admin team",
			user: &models.CreatedTeamAPIKey{
				ID: uuid.New(),
				Team: &models.Team{
					Name: models.AdminTeamName,
				},
			},
			expectTeam: models.AdminTeam(),
		},
		{
			name: "non-admin team is preserved",
			user: &models.CreatedTeamAPIKey{
				ID: uuid.New(),
				Team: &models.Team{
					ID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name: "team-a",
				},
			},
			expectTeam: &models.Team{
				ID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				Name: "team-a",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TeamForKey(tt.user)
			assert.Equal(t, tt.expectTeam, got)
		})
	}
}
