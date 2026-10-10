package platform

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCRDDefaultStability(t *testing.T) {
	for _, tc := range []struct{ name, schema, before, after, failure string }{
		{"missing", `{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}`, ``, `{}`, "$.spec.mode"},
		{"missing false default", `{"type":"object","properties":{"enabled":{"type":"boolean","default":false}}}`, ``, `{}`, "$.spec.enabled"},
		{"missing zero default", `{"type":"object","properties":{"count":{"type":"integer","default":0}}}`, ``, `{}`, "$.spec.count"},
		{"missing empty string default", `{"type":"object","properties":{"name":{"type":"string","default":""}}}`, ``, `{}`, "$.spec.name"},
		{"explicit override", `{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}`, ``, `{"mode":"custom"}`, ""},
		{"explicit zero values", `{"type":"object","properties":{"enabled":{"type":"boolean","default":true},"count":{"type":"integer","default":1},"name":{"type":"string","default":"name"}}}`, ``, `{"enabled":false,"count":0,"name":""}`, ""},
		{"nullable", `{"type":"object","properties":{"mode":{"type":"string","nullable":true,"default":"safe"}}}`, ``, `{"mode":null}`, ""},
		{"null default is not materialized", `{"type":"object","properties":{"mode":{"type":"string","nullable":true,"default":null}}}`, ``, `{}`, ""},
		{"non nullable", `{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}`, ``, `{"mode":null}`, "null is not allowed"},
		{"preserved parent known null", `{"type":"object","x-kubernetes-preserve-unknown-fields":true,"properties":{"mode":{"type":"string","default":"safe"}}}`, ``, `{"mode":null}`, "$.spec.mode: null is not allowed"},
		{"preserved parent known nullable", `{"type":"object","x-kubernetes-preserve-unknown-fields":true,"properties":{"mode":{"type":"string","nullable":true,"default":"safe"}}}`, ``, `{"mode":null,"opaque":{"unknown":true}}`, ""},
		{"preserved parent known omission", `{"type":"object","x-kubernetes-preserve-unknown-fields":true,"properties":{"mode":{"type":"string","default":"safe"}}}`, ``, `{"opaque":{"unknown":true}}`, "$.spec.mode"},
		{"absent parent", `{"type":"object","properties":{"child":{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}}}`, ``, `{}`, ""},
		{"present parent", `{"type":"object","properties":{"child":{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}}}`, ``, `{"child":{}}`, "$.spec.child.mode"},
		{"parent default", `{"type":"object","properties":{"child":{"type":"object","default":{},"properties":{"mode":{"type":"string","default":"safe"}}}}}`, ``, `{}`, "$.spec.child"},
		{"map values", `{"type":"object","additionalProperties":{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}}`, ``, `{"chosen":{}}`, "$.spec.chosen.mode"},
		{"list values", `{"type":"array","items":{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}}`, ``, `[{}]`, "$.spec[0].mode"},
		{"unchanged legacy omission", `{"type":"object","properties":{"mode":{"type":"string","default":"safe"},"label":{"type":"string"}}}`, `{}`, `{"label":"new"}`, ""},
		{"remove explicit value", `{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}`, `{"mode":"custom"}`, `{}`, "$.spec.mode"},
		{"unknown field", `{"type":"object","properties":{"mode":{"type":"string","default":"safe"}}}`, ``, `{"mode":"safe","typo":true}`, "unknown field typo"},
		{"new array item", `{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"mode":{"type":"string","default":"safe"}}}}`, `[{"id":"old"}]`, `[{"id":"new"},{"id":"old"}]`, "$.spec[0].mode"},
		{"reorder exact legacy items", `{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"mode":{"type":"string","default":"safe"}}}}`, `[{"id":"a"},{"id":"b"}]`, `[{"id":"b"},{"id":"a"}]`, ""},
		{"duplicated legacy item is new", `{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"mode":{"type":"string","default":"safe"}}}}`, `[{"id":"a"}]`, `[{"id":"a"},{"id":"a"}]`, "$.spec[1].mode"},
		{"changed array item", `{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"label":{"type":"string"},"mode":{"type":"string","default":"safe"}}}}`, `[{"id":"a"}]`, `[{"id":"a","label":"changed"}]`, "$.spec[0].mode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decode := func(raw string) any {
				var v any
				if err := json.Unmarshal([]byte(raw), &v); err != nil {
					t.Fatal(err)
				}
				return v
			}
			definition := customDefinition("Namespaced")
			schema := mapping(field(mapping(slice(field(definition, "spec", "versions"))[0]), "schema", "openAPIV3Schema"))
			mapping(schema["properties"])["spec"] = decode(tc.schema)
			model, err := newResourceModel(registryFixture(t), "1.37.0", []Object{definition})
			if err != nil {
				t.Fatal(err)
			}
			after := Object{"apiVersion": "example.test/v1", "kind": "Widget", "metadata": Object{"name": "one", "namespace": "tenant"}, "spec": decode(tc.after)}
			var before Object
			if tc.before != "" {
				before = clone(after)
				before["spec"] = decode(tc.before)
			}
			pristine := []Object{clone(definition), clone(after)}
			var old Object
			if before != nil {
				old = clone(before)
			}
			err = model.ValidateCRDDefaultStability(before, after)
			if tc.failure == "" && err != nil || tc.failure != "" && (err == nil || !strings.Contains(err.Error(), tc.failure)) {
				t.Fatalf("wanted %q, got %v", tc.failure, err)
			}
			if !reflect.DeepEqual(pristine, []Object{definition, after}) || !reflect.DeepEqual(before, old) {
				t.Fatal("validator mutated schema or input")
			}
			if tc.failure != "" {
				again := model.ValidateCRDDefaultStability(before, after)
				if again == nil || err.Error() != again.Error() {
					t.Fatal("non-deterministic error")
				}
			}
		})
	}
}
