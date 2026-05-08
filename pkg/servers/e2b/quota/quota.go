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

// Package quota provides per-team sandbox quota management backed by a Kubernetes ConfigMap.
//
// The ConfigMap (default name: "sandbox-quota") lives in the sandbox-manager's system namespace.
// Each key in ConfigMap.Data is a team name; the value is the integer maximum number of
// concurrently active sandboxes allowed for that team.
//
// Example ConfigMap:
//
//	apiVersion: v1
//	kind: ConfigMap
//	metadata:
//	  name: sandbox-quota
//	  namespace: <system-namespace>
//	data:
//	  team-a: "100"
//	  team-b: "50"
package quota

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ConfigMapName = "sandbox-quota"

// Storage reads and writes team quota configuration from a Kubernetes ConfigMap.
type Storage struct {
	client    client.Client
	namespace string
}

// NewStorage creates a Storage backed by the given client and system namespace.
func NewStorage(c client.Client, namespace string) *Storage {
	return &Storage{client: c, namespace: namespace}
}

// GetTeamMaxSandboxes returns the configured maximum number of active sandboxes for teamName.
// Returns (0, false, nil) when no quota has been set for the team.
func (s *Storage) GetTeamMaxSandboxes(ctx context.Context, teamName string) (int, bool, error) {
	cm := &corev1.ConfigMap{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: ConfigMapName, Namespace: s.namespace}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("get quota configmap: %w", err)
	}
	if cm.Data == nil {
		return 0, false, nil
	}
	raw, ok := cm.Data[teamName]
	if !ok {
		return 0, false, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, false, fmt.Errorf("invalid quota value %q for team %s", raw, teamName)
	}
	return n, true, nil
}

// SetTeamMaxSandboxes upserts the quota for teamName in the ConfigMap.
func (s *Storage) SetTeamMaxSandboxes(ctx context.Context, teamName string, maxSandboxes int) error {
	if maxSandboxes < 0 {
		return fmt.Errorf("maxSandboxes must be >= 0")
	}
	cm := &corev1.ConfigMap{}
	err := s.client.Get(ctx, types.NamespacedName{Name: ConfigMapName, Namespace: s.namespace}, cm)
	if apierrors.IsNotFound(err) {
		cm = &corev1.ConfigMap{}
		cm.Name = ConfigMapName
		cm.Namespace = s.namespace
		cm.Data = map[string]string{teamName: strconv.Itoa(maxSandboxes)}
		return s.client.Create(ctx, cm)
	}
	if err != nil {
		return fmt.Errorf("get quota configmap: %w", err)
	}
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[teamName] = strconv.Itoa(maxSandboxes)
	return s.client.Update(ctx, cm)
}

// DeleteTeamQuota removes the quota entry for teamName. It is a no-op when the team has no quota.
func (s *Storage) DeleteTeamQuota(ctx context.Context, teamName string) error {
	cm := &corev1.ConfigMap{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: ConfigMapName, Namespace: s.namespace}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get quota configmap: %w", err)
	}
	if cm.Data == nil {
		return nil
	}
	if _, ok := cm.Data[teamName]; !ok {
		return nil
	}
	delete(cm.Data, teamName)
	return s.client.Update(ctx, cm)
}
