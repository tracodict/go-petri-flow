package models

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ColorSet represents a type system for tokens in CPN
type ColorSet interface {
	// Name returns the name of the color set
	Name() string
	// IsMember checks if a value is a member of this color set
	IsMember(value interface{}) bool
	// IsTimed returns true if this color set supports timestamps
	IsTimed() bool
	// String returns a string representation of the color set
	String() string
}

// JsonMapColorSet represents arbitrary JSON objects (maps)
type JsonMapColorSet struct {
	name  string
	timed bool
}

// NewJsonMapColorSet creates a new jsonMap color set
func NewJsonMapColorSet(name string, timed bool) *JsonMapColorSet {
	return &JsonMapColorSet{
		name:  name,
		timed: timed,
	}
}

func (cs *JsonMapColorSet) Name() string {
	return cs.name
}

func (cs *JsonMapColorSet) IsMember(value interface{}) bool {
	// Accept any map[string]interface{} or map[any]any (from JSON unmarshaling)
	if value == nil {
		return false
	}
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Map {
		return true
	}
	// Accept also if value is a struct that can be marshaled to JSON (optional)
	return false
}

func (cs *JsonMapColorSet) IsTimed() bool {
	return cs.timed
}

func (cs *JsonMapColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}
	return fmt.Sprintf("colset %s = map%s", cs.name, timedStr)
}

// IntegerColorSet represents integer values
type IntegerColorSet struct {
	name   string
	timed  bool
	minVal *int
	maxVal *int
}

// NewIntegerColorSet creates a new integer color set
func NewIntegerColorSet(name string, timed bool) *IntegerColorSet {
	return &IntegerColorSet{
		name:  name,
		timed: timed,
	}
}

// NewIntegerColorSetWithRange creates a new integer color set with min/max bounds
func NewIntegerColorSetWithRange(name string, timed bool, minVal, maxVal int) *IntegerColorSet {
	return &IntegerColorSet{
		name:   name,
		timed:  timed,
		minVal: &minVal,
		maxVal: &maxVal,
	}
}

func (cs *IntegerColorSet) Name() string {
	return cs.name
}

func (cs *IntegerColorSet) IsMember(value interface{}) bool {
	switch v := value.(type) {
	case int:
		if cs.minVal != nil && v < *cs.minVal {
			return false
		}
		if cs.maxVal != nil && v > *cs.maxVal {
			return false
		}
		return true
	case int32:
		return cs.IsMember(int(v))
	case int64:
		return cs.IsMember(int(v))
	case float64:
		// Check if it's actually an integer
		if v == float64(int(v)) {
			return cs.IsMember(int(v))
		}
	}
	return false
}

func (cs *IntegerColorSet) IsTimed() bool {
	return cs.timed
}

func (cs *IntegerColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}
	rangeStr := ""
	if cs.minVal != nil && cs.maxVal != nil {
		rangeStr = fmt.Sprintf(" [%d..%d]", *cs.minVal, *cs.maxVal)
	}
	return fmt.Sprintf("colset %s = int%s%s", cs.name, rangeStr, timedStr)
}

// StringColorSet represents string values
type StringColorSet struct {
	name  string
	timed bool
}

// NewStringColorSet creates a new string color set
func NewStringColorSet(name string, timed bool) *StringColorSet {
	return &StringColorSet{
		name:  name,
		timed: timed,
	}
}

func (cs *StringColorSet) Name() string {
	return cs.name
}

func (cs *StringColorSet) IsMember(value interface{}) bool {
	_, ok := value.(string)
	return ok
}

func (cs *StringColorSet) IsTimed() bool {
	return cs.timed
}

func (cs *StringColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}
	return fmt.Sprintf("colset %s = string%s", cs.name, timedStr)
}

// BooleanColorSet represents boolean values
type BooleanColorSet struct {
	name  string
	timed bool
}

// NewBooleanColorSet creates a new boolean color set
func NewBooleanColorSet(name string, timed bool) *BooleanColorSet {
	return &BooleanColorSet{
		name:  name,
		timed: timed,
	}
}

func (cs *BooleanColorSet) Name() string {
	return cs.name
}

func (cs *BooleanColorSet) IsMember(value interface{}) bool {
	_, ok := value.(bool)
	return ok
}

func (cs *BooleanColorSet) IsTimed() bool {
	return cs.timed
}

func (cs *BooleanColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}
	return fmt.Sprintf("colset %s = bool%s", cs.name, timedStr)
}

// RealColorSet represents floating-point values
type RealColorSet struct {
	name  string
	timed bool
}

// NewRealColorSet creates a new real (float) color set
func NewRealColorSet(name string, timed bool) *RealColorSet {
	return &RealColorSet{
		name:  name,
		timed: timed,
	}
}

func (cs *RealColorSet) Name() string {
	return cs.name
}

func (cs *RealColorSet) IsMember(value interface{}) bool {
	switch value.(type) {
	case float32, float64:
		return true
	case int, int32, int64:
		return true // Integers can be promoted to reals
	}
	return false
}

func (cs *RealColorSet) IsTimed() bool {
	return cs.timed
}

func (cs *RealColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}
	return fmt.Sprintf("colset %s = real%s", cs.name, timedStr)
}

// UnitColorSet represents the unit type (single value)
type UnitColorSet struct {
	name  string
	timed bool
}

// NewUnitColorSet creates a new unit color set
func NewUnitColorSet(name string, timed bool) *UnitColorSet {
	return &UnitColorSet{
		name:  name,
		timed: timed,
	}
}

func (cs *UnitColorSet) Name() string {
	return cs.name
}

func (cs *UnitColorSet) IsMember(value interface{}) bool {
	// Unit type accepts nil or the string "unit"
	return value == nil || value == "unit" || value == "()"
}

func (cs *UnitColorSet) IsTimed() bool {
	return cs.timed
}

func (cs *UnitColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}
	return fmt.Sprintf("colset %s = unit%s", cs.name, timedStr)
}

// EnumeratedColorSet represents a finite set of named values
type EnumeratedColorSet struct {
	name   string
	timed  bool
	values []string
}

// NewEnumeratedColorSet creates a new enumerated color set
func NewEnumeratedColorSet(name string, timed bool, values []string) *EnumeratedColorSet {
	return &EnumeratedColorSet{
		name:   name,
		timed:  timed,
		values: values,
	}
}

func (cs *EnumeratedColorSet) Name() string {
	return cs.name
}

func (cs *EnumeratedColorSet) IsMember(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		return false
	}

	for _, v := range cs.values {
		if v == str {
			return true
		}
	}
	return false
}

func (cs *EnumeratedColorSet) IsTimed() bool {
	return cs.timed
}

func (cs *EnumeratedColorSet) GetValues() []string {
	return cs.values
}

func (cs *EnumeratedColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}
	valuesStr := strings.Join(cs.values, " | ")
	return fmt.Sprintf("colset %s = with %s%s", cs.name, valuesStr, timedStr)
}

// ProductColorSet represents tuples of values
type ProductColorSet struct {
	name       string
	timed      bool
	components []ColorSet
}

// NewProductColorSet creates a new product color set
func NewProductColorSet(name string, timed bool, components []ColorSet) *ProductColorSet {
	return &ProductColorSet{
		name:       name,
		timed:      timed,
		components: components,
	}
}

func (cs *ProductColorSet) Name() string {
	return cs.name
}

func (cs *ProductColorSet) IsMember(value interface{}) bool {
	// Check if value is a slice or array
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return false
	}

	// Check if the number of components matches
	if v.Len() != len(cs.components) {
		return false
	}

	// Check each component
	for i := 0; i < v.Len(); i++ {
		if !cs.components[i].IsMember(v.Index(i).Interface()) {
			return false
		}
	}

	return true
}

func (cs *ProductColorSet) IsTimed() bool {
	return cs.timed
}

func (cs *ProductColorSet) GetComponents() []ColorSet {
	return cs.components
}

func (cs *ProductColorSet) String() string {
	timedStr := ""
	if cs.timed {
		timedStr = " timed"
	}

	var componentNames []string
	for _, comp := range cs.components {
		componentNames = append(componentNames, comp.Name())
	}

	return fmt.Sprintf("colset %s = product %s%s", cs.name, strings.Join(componentNames, " * "), timedStr)
}

// Common color set instances
var (
	INT    = NewIntegerColorSet("INT", false)
	STRING = NewStringColorSet("STRING", false)
	BOOL   = NewBooleanColorSet("BOOL", false)
	REAL   = NewRealColorSet("REAL", false)
	UNIT   = NewUnitColorSet("UNIT", false)
)

// ParseColorSetValue attempts to parse a string value according to the color set
func ParseColorSetValue(colorSet ColorSet, valueStr string) (interface{}, error) {
	switch cs := colorSet.(type) {
	case *IntegerColorSet:
		if val, err := strconv.Atoi(valueStr); err == nil {
			if cs.IsMember(val) {
				return val, nil
			}
			return nil, fmt.Errorf("value %d is not in range for color set %s", val, cs.Name())
		}
		return nil, fmt.Errorf("cannot parse '%s' as integer", valueStr)

	case *RealColorSet:
		if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return val, nil
		}
		return nil, fmt.Errorf("cannot parse '%s' as real", valueStr)

	case *BooleanColorSet:
		if val, err := strconv.ParseBool(valueStr); err == nil {
			return val, nil
		}
		return nil, fmt.Errorf("cannot parse '%s' as boolean", valueStr)

	case *StringColorSet:
		return valueStr, nil

	case *UnitColorSet:
		if valueStr == "unit" || valueStr == "()" || valueStr == "" {
			return "unit", nil
		}
		return nil, fmt.Errorf("invalid unit value '%s'", valueStr)

	case *EnumeratedColorSet:
		if cs.IsMember(valueStr) {
			return valueStr, nil
		}
		return nil, fmt.Errorf("value '%s' is not a member of enumerated color set %s", valueStr, cs.Name())

	default:
		return nil, fmt.Errorf("unsupported color set type for parsing: %T", colorSet)
	}
}
