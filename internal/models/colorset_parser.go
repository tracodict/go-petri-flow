package models

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ColorSetParser handles parsing of color set definitions from strings
type ColorSetParser struct {
	colorSets   map[string]ColorSet           // Registry of defined color sets
	jsonSchemas map[string]*jsonschema.Schema // Compiled JSON Schemas
}

// NewColorSetParser creates a new color set parser
func NewColorSetParser() *ColorSetParser {
	parser := &ColorSetParser{
		colorSets:   make(map[string]ColorSet),
		jsonSchemas: make(map[string]*jsonschema.Schema),
	}

	// Register built-in color sets
	parser.RegisterColorSet(INT)
	parser.RegisterColorSet(STRING)
	parser.RegisterColorSet(BOOL)
	parser.RegisterColorSet(REAL)
	parser.RegisterColorSet(UNIT)

	return parser
}

// RegisterColorSet registers a color set in the parser
func (p *ColorSetParser) RegisterColorSet(colorSet ColorSet) {
	p.colorSets[colorSet.Name()] = colorSet
}

// GetColorSet retrieves a registered color set by name
func (p *ColorSetParser) GetColorSet(name string) (ColorSet, bool) {
	cs, exists := p.colorSets[name]
	return cs, exists
}

// ParseColorSetDefinition parses a color set definition string
// Examples:
//
//	"colset INT = int;"
//	"colset MyInt = int timed;"
//	"colset Color = with red | green | blue;"
//	"colset Pair = product INT * STRING;"
func (p *ColorSetParser) ParseColorSetDefinition(definition string) (ColorSet, error) {
	// Remove extra whitespace and normalize
	definition = strings.TrimSpace(definition)
	if !strings.HasSuffix(definition, ";") {
		definition += ";"
	}

	// Basic regex to parse color set definitions
	// colset NAME = TYPE [timed];
	re := regexp.MustCompile(`^colset\s+(\w+)\s*=\s*(.+?)\s*;?\s*$`)
	matches := re.FindStringSubmatch(definition)

	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid color set definition format: %s", definition)
	}

	name := matches[1]
	typeDefinition := strings.TrimSpace(matches[2])

	// Check if it's timed
	timed := false
	if strings.HasSuffix(typeDefinition, " timed") {
		timed = true
		typeDefinition = strings.TrimSuffix(typeDefinition, " timed")
		typeDefinition = strings.TrimSpace(typeDefinition)
	}

	colorSet, err := p.parseTypeDefinition(name, typeDefinition, timed)
	if err != nil {
		return nil, fmt.Errorf("error parsing type definition '%s': %v", typeDefinition, err)
	}

	// Register the new color set
	p.RegisterColorSet(colorSet)

	return colorSet, nil
}

// parseTypeDefinition parses the type part of a color set definition
func (p *ColorSetParser) parseTypeDefinition(name, typeDefinition string, timed bool) (ColorSet, error) {
	typeDefinition = strings.TrimSpace(typeDefinition)

	switch {
	case typeDefinition == "int":
		return NewIntegerColorSet(name, timed), nil
	case typeDefinition == "string":
		return NewStringColorSet(name, timed), nil
	case typeDefinition == "bool":
		return NewBooleanColorSet(name, timed), nil
	case typeDefinition == "real":
		return NewRealColorSet(name, timed), nil
	case typeDefinition == "unit":
		return NewUnitColorSet(name, timed), nil
	case typeDefinition == "map":
		// deprecated alias -> json (untyped)
		log.Printf("[DEPRECATION] color set type 'map' is deprecated; use 'json' instead")
		return NewJsonColorSet(name, timed, "", nil), nil
	case typeDefinition == "json":
		return NewJsonColorSet(name, timed, "", nil), nil
	case strings.HasPrefix(typeDefinition, "json<") && strings.HasSuffix(typeDefinition, ">"):
		schemaName := strings.TrimSuffix(strings.TrimPrefix(typeDefinition, "json<"), ">")
		schema, ok := p.jsonSchemas[schemaName]
		if !ok {
			return nil, fmt.Errorf("unknown json schema '%s'", schemaName)
		}
		return NewJsonColorSet(name, timed, schemaName, schema), nil
	case strings.HasPrefix(typeDefinition, "int[") && strings.HasSuffix(typeDefinition, "]"):
		return p.parseIntegerRange(name, typeDefinition, timed)
	case strings.HasPrefix(typeDefinition, "with "):
		return p.parseEnumerated(name, typeDefinition, timed)
	case strings.HasPrefix(typeDefinition, "product "):
		return p.parseProduct(name, typeDefinition, timed)
	default:
		// Check if it's a reference to an existing color set
		if existingCS, exists := p.GetColorSet(typeDefinition); exists {
			// Create a new color set with the same type but different name and timed property
			return p.cloneColorSetWithNewName(existingCS, name, timed)
		}
		return nil, fmt.Errorf("unknown type definition: %s", typeDefinition)
	}
}

// parseIntegerRange parses integer range definitions like "int[1..10]"
func (p *ColorSetParser) parseIntegerRange(name, typeDefinition string, timed bool) (ColorSet, error) {
	// Extract range from int[min..max]
	re := regexp.MustCompile(`^int\[(\d+)\.\.(\d+)\]$`)
	matches := re.FindStringSubmatch(typeDefinition)

	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid integer range format: %s", typeDefinition)
	}

	minVal, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minimum value: %s", matches[1])
	}

	maxVal, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid maximum value: %s", matches[2])
	}

	if minVal > maxVal {
		return nil, fmt.Errorf("minimum value %d is greater than maximum value %d", minVal, maxVal)
	}

	return NewIntegerColorSetWithRange(name, timed, minVal, maxVal), nil
}

// parseEnumerated parses enumerated definitions like "with red | green | blue"
func (p *ColorSetParser) parseEnumerated(name, typeDefinition string, timed bool) (ColorSet, error) {
	// Remove "with " prefix
	valuesPart := strings.TrimPrefix(typeDefinition, "with ")
	valuesPart = strings.TrimSpace(valuesPart)

	// Split by |
	values := strings.Split(valuesPart, "|")

	// Trim whitespace from each value
	for i, value := range values {
		values[i] = strings.TrimSpace(value)
		if values[i] == "" {
			return nil, fmt.Errorf("empty value in enumerated color set")
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("enumerated color set must have at least one value")
	}

	return NewEnumeratedColorSet(name, timed, values), nil
}

// parseProduct parses product definitions like "product INT * STRING"
func (p *ColorSetParser) parseProduct(name, typeDefinition string, timed bool) (ColorSet, error) {
	// Remove "product " prefix
	componentsPart := strings.TrimPrefix(typeDefinition, "product ")
	componentsPart = strings.TrimSpace(componentsPart)

	// Split by *
	componentNames := strings.Split(componentsPart, "*")

	// Trim whitespace and resolve component color sets
	var components []ColorSet
	for _, componentName := range componentNames {
		componentName = strings.TrimSpace(componentName)
		if componentName == "" {
			return nil, fmt.Errorf("empty component in product color set")
		}

		component, exists := p.GetColorSet(componentName)
		if !exists {
			return nil, fmt.Errorf("unknown color set component: %s", componentName)
		}

		components = append(components, component)
	}

	if len(components) < 2 {
		return nil, fmt.Errorf("product color set must have at least two components")
	}

	return NewProductColorSet(name, timed, components), nil
}

// cloneColorSetWithNewName creates a new color set based on an existing one but with different name and timed property
func (p *ColorSetParser) cloneColorSetWithNewName(original ColorSet, newName string, timed bool) (ColorSet, error) {
	switch cs := original.(type) {
	case *IntegerColorSet:
		if cs.minVal != nil && cs.maxVal != nil {
			return NewIntegerColorSetWithRange(newName, timed, *cs.minVal, *cs.maxVal), nil
		}
		return NewIntegerColorSet(newName, timed), nil

	case *StringColorSet:
		return NewStringColorSet(newName, timed), nil

	case *BooleanColorSet:
		return NewBooleanColorSet(newName, timed), nil

	case *RealColorSet:
		return NewRealColorSet(newName, timed), nil

	case *UnitColorSet:
		return NewUnitColorSet(newName, timed), nil

	case *EnumeratedColorSet:
		return NewEnumeratedColorSet(newName, timed, cs.GetValues()), nil

	case *ProductColorSet:
		return NewProductColorSet(newName, timed, cs.GetComponents()), nil

	default:
		return nil, fmt.Errorf("unsupported color set type for cloning: %T", original)
	}
}

// ParseMultipleDefinitions parses multiple color set definitions separated by newlines
func (p *ColorSetParser) ParseMultipleDefinitions(definitions string) ([]ColorSet, error) {
	lines := strings.Split(definitions, "\n")
	var colorSets []ColorSet

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		colorSet, err := p.ParseColorSetDefinition(line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line %d: %v", i+1, err)
		}

		colorSets = append(colorSets, colorSet)
	}

	return colorSets, nil
}

// GetAllColorSets returns all registered color sets
func (p *ColorSetParser) GetAllColorSets() map[string]ColorSet {
	result := make(map[string]ColorSet)
	for name, cs := range p.colorSets {
		result[name] = cs
	}
	return result
}
