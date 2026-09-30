package main

import (
	"encoding/json"
	"testing"
)

func podFixture(t *testing.T) []runtimePod {
	t.Helper()
	var pods []runtimePod
	err := json.Unmarshal([]byte(`[
 {"metadata":{"name":"web-1","namespace":"workload-web"},"spec":{"nodeName":"test-worker3"},"status":{"phase":"Running","conditions":[{"type":"Ready","status":"True"}]}},
 {"metadata":{"name":"envoy-atlas-gateway-1","namespace":"envoy-gateway-system","labels":{"app.kubernetes.io/component":"proxy","gateway.envoyproxy.io/owning-gateway-name":"development","gateway.envoyproxy.io/owning-gateway-namespace":"atlas-gateway"}},"spec":{"nodeName":"test-worker"},"status":{"phase":"Running","conditions":[{"type":"Ready","status":"True"}]}}
 ]`), &pods)
	if err != nil {
		t.Fatal(err)
	}
	return pods
}

func TestEveryReplicaMustUseItsRoleNodeRegardlessOfOrder(t *testing.T) {
	if err := verifyPods(podFixture(t), "test"); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, 1} {
		for _, first := range []bool{false, true} {
			pods := podFixture(t)
			extra := pods[index]
			extra.Metadata.Name += "-second"
			// Multiple correctly placed replicas are valid.
			if err := verifyPods(append(pods, extra), "test"); err != nil {
				t.Fatal(err)
			}
			extra.Spec.NodeName = "test-worker2"
			if first {
				pods = append([]runtimePod{extra}, pods...)
			} else {
				pods = append(pods, extra)
			}
			if err := verifyPods(pods, "test"); err == nil {
				t.Fatalf("misplaced replica masked: index=%d first=%v", index, first)
			}
		}
	}
}

func TestReadinessAndActualGatewayIdentityRequired(t *testing.T) {
	for _, change := range []func([]runtimePod){
		func(p []runtimePod) { p[0].Status.Phase = "Pending" },
		func(p []runtimePod) { p[0].Status.Conditions = nil },
		func(p []runtimePod) { p[1].Metadata.Namespace = "other" },
		func(p []runtimePod) { p[1].Metadata.Labels["gateway.envoyproxy.io/owning-gateway-name"] = "other" },
		func(p []runtimePod) { p[1].Metadata.Labels["app.kubernetes.io/component"] = "controller" },
	} {
		pods := podFixture(t)
		change(pods)
		if err := verifyPods(pods, "test"); err == nil {
			t.Fatal("incomplete runtime proof accepted")
		}
	}
	if err := verifyPods(nil, "test"); err == nil {
		t.Fatal("empty inventory accepted")
	}
	pods := podFixture(t)
	job := runtimePod{}
	job.Status.Phase = "Succeeded"
	if err := verifyPods(append(pods, job), "test"); err != nil {
		t.Fatal("completed job must not require readiness", err)
	}
}
