// Copyright 2026 irulast
// SPDX-License-Identifier: AGPL-3.0-or-later

package controller

import (
	"testing"

	"github.com/minio/minio-go/v7/pkg/set"
	miniov2 "github.com/minio/operator/pkg/apis/minio.min.io/v2"
	miniofake "github.com/minio/operator/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

// A scoped controller must not fetch credentials, update status, or enqueue
// health work for a normal Tenant in another namespace. Fake clients keep this
// check entirely in memory; it never talks to a cluster or MinIO endpoint.
func TestScopedHealthWorkers(t *testing.T) {
	outside := &miniov2.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: "outside"},
	}
	outside.Status.Pools = []miniov2.PoolStatus{{State: miniov2.PoolInitialized}}
	kube := kubefake.NewSimpleClientset()
	minio := miniofake.NewSimpleClientset(outside)
	c := &Controller{namespacesToWatch: set.NewStringSet("inside"), kubeClientSet: kube, minioClientSet: minio}
	if err := c.tenantsHealthMonitor(); err != nil {
		t.Fatal(err)
	}
	if got, err := c.updateHealthStatusForTenant(outside); err != nil || got != nil {
		t.Fatalf("out-of-scope health result = %v, %v", got, err)
	}
	if _, err := c.syncHealthCheckHandler("outside/store"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.syncHandler("outside/store"); err != nil {
		t.Fatal(err)
	}
	c.handlePodChange(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "store-0", Namespace: "outside", Labels: map[string]string{miniov2.TenantLabel: "store"},
	}})
	if actions := kube.Actions(); len(actions) != 0 {
		t.Fatalf("unexpected Kubernetes actions outside watch scope: %v", actions)
	}
	actions := minio.Actions()
	if len(actions) != 1 || actions[0].GetVerb() != "list" || actions[0].GetResource().Resource != "tenants" {
		t.Fatalf("expected only the global tenant inventory read, got %v", actions)
	}
}

func TestHealthScopeIncludesConfiguredAndUnrestrictedTenants(t *testing.T) {
	tenant := &miniov2.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "new-store", Namespace: "inside"}}
	for _, scope := range []set.StringSet{nil, set.NewStringSet("inside", "another")} {
		c := &Controller{namespacesToWatch: scope}
		got, err := c.updateHealthStatusForTenant(tenant)
		if err != nil || got != tenant {
			t.Fatalf("normal uninitialized tenant was excluded: %v, %v", got, err)
		}
	}
}
