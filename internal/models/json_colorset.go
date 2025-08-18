package models

import (
	"fmt"
	"reflect"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// JsonColorSet represents JSON (object/array) values optionally validated by a JSON Schema.
// Deprecated: previous JsonMapColorSet should be replaced by this; parser still accepts 'map' as alias for backward compatibility.
type JsonColorSet struct {
	name       string
	timed      bool
	schemaName string
	schema     *jsonschema.Schema
}

func NewJsonColorSet(name string, timed bool, schemaName string, schema *jsonschema.Schema) *JsonColorSet {
	return &JsonColorSet{name: name, timed: timed, schemaName: schemaName, schema: schema}
}

func (cs *JsonColorSet) Name() string { return cs.name }

func (cs *JsonColorSet) IsTimed() bool { return cs.timed }

func (cs *JsonColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}
	if cs.schemaName != "" {
		return fmt.Sprintf("colset %s = json<%s>%s", cs.name, cs.schemaName, timedStr)
	}
	return fmt.Sprintf("colset %s = json%s", cs.name, timedStr)
}

// IsMember validates the value against schema (if present). Accept maps and slices.
func (cs *JsonColorSet) IsMember(value interface{}) bool {
	if value == nil {
		return false
	}
	kind := reflect.TypeOf(value).Kind()
	if kind != reflect.Map && kind != reflect.Slice && kind != reflect.Array {
		return false
	}
	if cs.schema == nil {
		return true
	}
	if err := cs.schema.Validate(value); err != nil {
		return false
	}
	return true
}

// Validate returns a detailed error if the value does not conform.
func (cs *JsonColorSet) Validate(value interface{}) error {
	if value == nil {
		return fmt.Errorf("nil value not allowed")
	}
	kind := reflect.TypeOf(value).Kind()
	if kind != reflect.Map && kind != reflect.Slice && kind != reflect.Array {
		return fmt.Errorf("expected object or array, got %s", kind.String())
	}
	if cs.schema == nil {
		return nil
	}
	if err := cs.schema.Validate(value); err != nil {
		return err
	}
	return nil
}
