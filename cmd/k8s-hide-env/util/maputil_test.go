package util

import (
	"reflect"
	"testing"
)

func TestExtractMapList(t *testing.T) {
	tests := []struct {
		testCase string
		testMap  map[string]interface{}
		key      string
		expected interface{}
	}{
		{
			"map is nil",
			nil,
			"entryName",
			nil,
		},
		{
			"map is empty",
			map[string]interface{}{},
			"entryName",
			nil,
		},
		{
			"map does not contain list",
			map[string]interface{}{"entryName": "value"},
			"entryName",
			nil,
		},
		{
			"happy path",
			map[string]interface{}{"entryName": []map[string]interface{}{{"entryName": "value"}}},
			"entryName",
			[]map[string]interface{}{{"entryName": "value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			actual := ExtractMapList(tt.testMap, tt.key)
			if actual == nil && tt.expected == nil {
				return
			}
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("Key value mismatch: %s => %s", tt.expected, actual)
			}
		})
	}
}

func TestExtractStringList(t *testing.T) {
	tests := []struct {
		testCase string
		testMap  map[string]interface{}
		key      string
		expected interface{}
	}{
		{
			"map is nil",
			nil,
			"entryName",
			nil,
		},
		{
			"map is empty",
			map[string]interface{}{},
			"entryName",
			nil,
		},
		{
			"map does not contain list",
			map[string]interface{}{"entryName": "value"},
			"entryName",
			nil,
		},
		{
			"happy path",
			map[string]interface{}{"entryName": []string{"str1", "str2"}},
			"entryName",
			[]string{"str1", "str2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			actual := ExtractStringList(tt.testMap, tt.key)
			if actual == nil && tt.expected == nil {
				return
			}
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("Key value mismatch: %s => %s", tt.expected, actual)
			}
		})
	}
}

func TestExtractMap(t *testing.T) {
	tests := []struct {
		testCase string
		testMap  map[string]interface{}
		key      string
		expected map[string]interface{}
	}{
		{
			"map is nil",
			nil,
			"entryName",
			nil,
		},
		{
			"map is empty",
			map[string]interface{}{},
			"entryName",
			nil,
		},
		{
			"map does not contain list",
			map[string]interface{}{"entryName": "value"},
			"entryName",
			nil,
		},
		{
			"happy path",
			map[string]interface{}{"entryName": map[string]interface{}{"entryName": "value"}},
			"entryName",
			map[string]interface{}{"entryName": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			actual := ExtractMap(tt.testMap, tt.key)
			if actual == nil && tt.expected == nil {
				return
			}
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("Key value mismatch: %s => %s", tt.expected, actual)
			}
		})
	}
}

func TestGetArrayIndex(t *testing.T) {
	tests := []struct {
		testCase  string
		testMap   []map[string]interface{}
		entryName string
		expected  int
	}{
		{
			"map is nil",
			nil,
			"my-name",
			-1,
		},
		{
			"map is empty",
			[]map[string]interface{}{},
			"my-name",
			-1,
		},
		{
			"map does not contain entryName",
			[]map[string]interface{}{{"name": "my-name"}},
			"wrong-entryName",
			-1,
		},
		{
			"happy path",
			[]map[string]interface{}{{"name": "my-name"}},
			"my-name",
			0,
		},
		{
			"happy path",
			[]map[string]interface{}{{"name": "my-name"}},
			"my-name",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			actual := GetArrayIndex(tt.testMap, tt.entryName)
			if tt.expected != actual {
				t.Errorf("Index mismatch: %d => %d", tt.expected, actual)
			}
		})
	}
}
