package main

import (
	"errors"
	"fmt"
	"strings"
)

type runtimePod struct {
	Metadata struct {
		Name, Namespace string
		Labels          map[string]string
	}
	Spec   struct{ NodeName string }
	Status struct {
		Phase      string
		Conditions []struct{ Type, Status string }
	}
}

func verifyPods(pods []runtimePod, cluster string) error {
	web, proxy := false, false
	for _, p := range pods {
		if p.Status.Phase == "Succeeded" {
			continue
		}
		ready := false
		for _, c := range p.Status.Conditions {
			if c.Type == "Ready" && c.Status == "True" {
				ready = true
			}
		}
		if p.Status.Phase != "Running" || !ready {
			return fmt.Errorf("pod not Ready: %s/%s", p.Metadata.Namespace, p.Metadata.Name)
		}
		if p.Metadata.Namespace == "workload-web" {
			if p.Spec.NodeName != cluster+"-worker3" {
				return fmt.Errorf("workload Pod outside data node: %s", p.Metadata.Name)
			}
			web = true
		}
		if p.Metadata.Namespace == "envoy-gateway-system" && strings.HasPrefix(p.Metadata.Name, "envoy-atlas-gateway-") &&
			p.Metadata.Labels["app.kubernetes.io/component"] == "proxy" &&
			p.Metadata.Labels["gateway.envoyproxy.io/owning-gateway-name"] == "development" &&
			p.Metadata.Labels["gateway.envoyproxy.io/owning-gateway-namespace"] == "atlas-gateway" {
			if p.Spec.NodeName != cluster+"-worker" {
				return fmt.Errorf("gateway Pod outside gateway node: %s", p.Metadata.Name)
			}
			proxy = true
		}
	}
	if !web || !proxy {
		return errors.New("gateway/data placement failed")
	}
	return nil
}
