package query

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// toSnakeCase converts PascalCase/camelCase to snake_case.
// e.g. "CreatedAt" → "created_at", "HTTPServer" → "http_server"
func toSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				// Insert underscore before uppercase that follows lowercase,
				// or before uppercase that starts a new word in a run of capitals.
				if unicode.IsLower(prev) || (i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// structToMap converts a struct to map[string]any using db tags with snake_case fallback.
// Skips unexported fields and fields tagged with db:"-".
func structToMap(v any) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("structToMap expects a struct, got %T", v)
	}

	rt := rv.Type()
	m := make(map[string]any, rt.NumField())

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check db tag
		tag := field.Tag.Get("db")
		if tag == "-" {
			continue
		}

		// Use tag value or fall back to snake_case
		colName := tag
		if colName == "" {
			colName = toSnakeCase(field.Name)
		}

		m[colName] = rv.Field(i).Interface()
	}

	return m, nil
}

// structsToMaps converts a slice of structs to []map[string]any.
func structsToMaps(v any) ([]map[string]any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("structsToMaps expects a slice, got %T", v)
	}

	if rv.Len() == 0 {
		return nil, nil
	}

	maps := make([]map[string]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i).Interface()
		m, err := structToMap(elem)
		if err != nil {
			return nil, err
		}
		maps[i] = m
	}
	return maps, nil
}
