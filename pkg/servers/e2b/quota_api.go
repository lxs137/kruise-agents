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

package e2b

import (
	"context"
	"encoding/json"
	"net/http"

	"k8s.io/klog/v2"

	"github.com/openkruise/agents/pkg/sandbox-manager/infra"
	"github.com/openkruise/agents/pkg/servers/e2b/keys"
	"github.com/openkruise/agents/pkg/servers/e2b/models"
	"github.com/openkruise/agents/pkg/servers/web"
)

// GetTeamQuota returns the quota configuration and current active sandbox count for a team.
// Admin keys can query any team; non-admin keys can only query their own team.
func (sc *Controller) GetTeamQuota(r *http.Request) (web.ApiResponse[*models.TeamQuotaStatus], *web.ApiError) {
	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusUnauthorized,
			Message: "User not found",
		}
	}

	teamName := r.PathValue("teamName")
	if teamName == "" {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusBadRequest,
			Message: "teamName is required",
		}
	}

	callerTeam := keys.TeamForKey(user)
	if callerTeam.Name != models.AdminTeamName && callerTeam.Name != teamName {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusForbidden,
			Message: "you are not allowed to view quota for another team",
		}
	}

	status, apiErr := sc.buildTeamQuotaStatus(ctx, teamName)
	if apiErr != nil {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, apiErr
	}
	return web.ApiResponse[*models.TeamQuotaStatus]{Body: status}, nil
}

// SetTeamQuota sets the maximum sandbox count for a team. Admin only.
func (sc *Controller) SetTeamQuota(r *http.Request) (web.ApiResponse[*models.TeamQuotaStatus], *web.ApiError) {
	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusUnauthorized,
			Message: "User not found",
		}
	}

	callerTeam := keys.TeamForKey(user)
	if callerTeam.Name != models.AdminTeamName {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusForbidden,
			Message: "only admin can set team quota",
		}
	}

	teamName := r.PathValue("teamName")
	if teamName == "" {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusBadRequest,
			Message: "teamName is required",
		}
	}

	var req models.TeamQuota
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}
	if req.MaxSandboxes < 0 {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusBadRequest,
			Message: "maxSandboxes must be >= 0",
		}
	}

	if err := sc.quota.SetTeamMaxSandboxes(ctx, teamName, req.MaxSandboxes); err != nil {
		klog.FromContext(ctx).Error(err, "failed to set team quota", "team", teamName)
		return web.ApiResponse[*models.TeamQuotaStatus]{}, &web.ApiError{
			Code:    http.StatusInternalServerError,
			Message: "failed to set team quota",
		}
	}

	status, apiErr := sc.buildTeamQuotaStatus(ctx, teamName)
	if apiErr != nil {
		return web.ApiResponse[*models.TeamQuotaStatus]{}, apiErr
	}
	return web.ApiResponse[*models.TeamQuotaStatus]{Body: status}, nil
}

// DeleteTeamQuota removes the quota limit for a team, making it unlimited. Admin only.
func (sc *Controller) DeleteTeamQuota(r *http.Request) (web.ApiResponse[struct{}], *web.ApiError) {
	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		return web.ApiResponse[struct{}]{}, &web.ApiError{
			Code:    http.StatusUnauthorized,
			Message: "User not found",
		}
	}

	callerTeam := keys.TeamForKey(user)
	if callerTeam.Name != models.AdminTeamName {
		return web.ApiResponse[struct{}]{}, &web.ApiError{
			Code:    http.StatusForbidden,
			Message: "only admin can delete team quota",
		}
	}

	teamName := r.PathValue("teamName")
	if teamName == "" {
		return web.ApiResponse[struct{}]{}, &web.ApiError{
			Code:    http.StatusBadRequest,
			Message: "teamName is required",
		}
	}

	if err := sc.quota.DeleteTeamQuota(ctx, teamName); err != nil {
		klog.FromContext(ctx).Error(err, "failed to delete team quota", "team", teamName)
		return web.ApiResponse[struct{}]{}, &web.ApiError{
			Code:    http.StatusInternalServerError,
			Message: "failed to delete team quota",
		}
	}
	return web.ApiResponse[struct{}]{Code: http.StatusNoContent}, nil
}

func (sc *Controller) buildTeamQuotaStatus(ctx context.Context, teamName string) (*models.TeamQuotaStatus, *web.ApiError) {
	max, configured, err := sc.quota.GetTeamMaxSandboxes(ctx, teamName)
	if err != nil {
		klog.FromContext(ctx).Error(err, "failed to read team quota", "team", teamName)
		return nil, &web.ApiError{
			Code:    http.StatusInternalServerError,
			Message: "failed to read team quota",
		}
	}
	if !configured {
		max = -1 // sentinel: no limit
	}

	sandboxes, err := sc.manager.GetInfra().SelectSandboxes(ctx, infra.SelectSandboxesOptions{Namespace: teamName})
	if err != nil {
		klog.FromContext(ctx).Error(err, "failed to list sandboxes for quota status", "team", teamName)
		return nil, &web.ApiError{
			Code:    http.StatusInternalServerError,
			Message: "failed to list sandboxes",
		}
	}

	return &models.TeamQuotaStatus{
		TeamName:        teamName,
		MaxSandboxes:    max,
		ActiveSandboxes: countActiveSandboxes(sandboxes),
	}, nil
}
