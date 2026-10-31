// Copyright 2026 irulast
// SPDX-License-Identifier: AGPL-3.0-or-later

package controller

import (
	miniov2 "github.com/minio/operator/pkg/apis/minio.min.io/v2"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestTenantImageUpdateMode(t *testing.T) {
	for _, tc := range []struct {
		name      string
		env       []corev1.EnvVar
		imageOnly bool
	}{
		{name: "legacy default"},
		{name: "explicit image updates", env: []corev1.EnvVar{{Name: "MINIO_UPDATE", Value: "off"}}, imageOnly: true},
		{name: "explicit binary updates", env: []corev1.EnvVar{{Name: "MINIO_UPDATE", Value: "on"}}},
		{name: "unresolved secret retains legacy mode", env: []corev1.EnvVar{{Name: "MINIO_UPDATE", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{Key: "update"}}}}},
		{name: "last configured value wins", env: []corev1.EnvVar{{Name: "MINIO_UPDATE", Value: "on"}, {Name: "MINIO_UPDATE", Value: "off"}}, imageOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tenant := &miniov2.Tenant{Spec: miniov2.TenantSpec{Env: tc.env}}
			if got := tenantDisablesInPlaceUpdate(tenant); got != tc.imageOnly {
				t.Fatalf("image-only mode = %v, want %v", got, tc.imageOnly)
			}
		})
	}
}
