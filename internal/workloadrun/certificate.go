package workloadrun

import (
	"encoding/json"
	"errors"
)

// cert-manager's Ready condition proves issuance of the referenced TLS Secret.
// It must describe the generation already checked for desired content and Argo
// ownership by observe; a stale Ready cannot authorize a Gateway listener.
func certificateReady(live Object, generation *int64) error {
	if generation == nil || *generation < 1 {
		return errors.New("TLS certificate generation missing or invalid")
	}
	var conditions []struct {
		Type               string `json:"type"`
		Status             string `json:"status"`
		ObservedGeneration *int64 `json:"observedGeneration"`
	}
	data, err := json.Marshal(at(live, "status", "conditions"))
	if err != nil || json.Unmarshal(data, &conditions) != nil {
		return errors.New("TLS certificate conditions malformed")
	}
	ready := false
	seen := false
	for _, c := range conditions {
		if c.Type == "" {
			return errors.New("TLS certificate condition type missing")
		}
		if c.Type != "Ready" {
			continue
		}
		if seen {
			return errors.New("TLS certificate Ready condition duplicated")
		}
		seen = true
		if c.Status != "True" && c.Status != "False" && c.Status != "Unknown" {
			return errors.New("TLS certificate Ready status invalid")
		}
		if c.ObservedGeneration == nil || *c.ObservedGeneration < 1 || *c.ObservedGeneration > *generation {
			return errors.New("TLS certificate Ready generation missing or invalid")
		}
		ready = c.Status == "True" && *c.ObservedGeneration == *generation
	}
	if !ready {
		return Pending("TLS certificate not Ready for current generation")
	}
	return nil
}
