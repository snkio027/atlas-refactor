package platform

import (
	"fmt"
	"reflect"
	"sort"
)

// ValidateCRDDefaultStability rejects omissions that the locked CRD schema
// would materialize in newly authored content. It never inserts defaults or
// consults a live API. Kubernetes built-ins, admission webhooks and CEL are not
// modeled here. Unchanged baseline subtrees retain their historical bytes.
func (m *ResourceModel) ValidateCRDDefaultStability(before, after Object) error {
	if err := m.Validate(after, ""); err != nil {
		return err
	}
	typ, err := m.Resolve(after)
	if err != nil {
		return err
	}
	if typ.Schema == nil {
		return nil
	}
	if before != nil && (before["apiVersion"] != after["apiVersion"] || identity(before) != identity(after)) {
		return fmt.Errorf("default stability baseline identity differs: %s", identity(after))
	}
	// These fields belong to Kubernetes/Argo runtime bookkeeping, not CR spec
	// defaulting. Nested metadata in a declared spec remains schema-checked.
	props := mapping(typ.Schema["properties"])
	keys := sortedDefaultKeys(props)
	for _, key := range keys {
		if key == "metadata" || key == "status" || key == "apiVersion" || key == "kind" {
			continue
		}
		old, oldExists := before[key]
		value, exists := after[key]
		if err := checkDefaultDelta(old, oldExists, before != nil, value, exists, mapping(props[key]), "$."+key); err != nil {
			return fmt.Errorf("%s: %w", identity(after), err)
		}
	}
	return nil
}

func sortedDefaultKeys(m Object) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func checkDefaultDelta(old any, oldExists, knownBaseline bool, value any, exists bool, schema Object, path string) error {
	if knownBaseline && oldExists == exists && reflect.DeepEqual(old, value) {
		return nil
	}
	if !exists {
		if schema["default"] != nil {
			return fmt.Errorf("%s: omitted CRD default in generated content; emit the intended value explicitly", path)
		}
		// An absent optional parent without its own default is not materialized.
		return nil
	}
	// Check declared nullable semantics even beneath preserve-unknown parents,
	// where the existing structural validator deliberately stops descending.
	if value == nil {
		if schema["nullable"] != true {
			return fmt.Errorf("%s: null is not allowed", path)
		}
		return nil
	}
	switch value := value.(type) {
	case map[string]any:
		previous, oldMap := old.(map[string]any)
		props := mapping(schema["properties"])
		for _, key := range sortedDefaultKeys(props) {
			prior, priorExists := previous[key]
			next, nextExists := value[key]
			if err := checkDefaultDelta(prior, priorExists, knownBaseline && oldExists && oldMap, next, nextExists, mapping(props[key]), path+"."+key); err != nil {
				return err
			}
		}
		if extra, ok := schema["additionalProperties"].(map[string]any); ok {
			for _, key := range sortedDefaultKeys(value) {
				if _, declared := props[key]; declared {
					continue
				}
				prior, priorExists := previous[key]
				if err := checkDefaultDelta(prior, priorExists, knownBaseline && oldExists && oldMap, value[key], true, extra, path+"."+key); err != nil {
					return err
				}
			}
		}
	case []any:
		previous, oldList := old.([]any)
		used := make([]bool, len(previous))
		for i, item := range value {
			unchanged := false
			if knownBaseline && oldExists && oldList {
				for j, prior := range previous {
					if !used[j] && reflect.DeepEqual(prior, item) {
						used[j] = true
						unchanged = true
						break
					}
				}
			}
			if unchanged {
				continue
			}
			// Array positions are not resource identities. New or changed items must
			// be complete; never inherit omissions from an unrelated old index.
			if err := checkDefaultDelta(nil, false, false, item, true, mapping(schema["items"]), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}
