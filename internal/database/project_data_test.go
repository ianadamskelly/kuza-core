package database

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateRecordData(t *testing.T) {
	schema := json.RawMessage(`{
		"required": ["name"],
		"fields": {
			"name": "text",
			"age": "number",
			"published": "boolean"
		}
	}`)
	data := json.RawMessage(`{"name":"Ian","age":42,"published":true}`)

	if err := validateRecordData(schema, data); err != nil {
		t.Fatalf("expected valid data, got %v", err)
	}
}

func TestValidateRecordDataMissingRequired(t *testing.T) {
	schema := json.RawMessage(`{"required":["name"]}`)
	data := json.RawMessage(`{"headline":"Educator"}`)

	err := validateRecordData(schema, data)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestValidateRecordDataWrongType(t *testing.T) {
	schema := json.RawMessage(`{"fields":{"age":"number"}}`)
	data := json.RawMessage(`{"age":"forty two"}`)

	err := validateRecordData(schema, data)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestValidateRecordDataRequiresObject(t *testing.T) {
	err := validateRecordData(json.RawMessage(`{}`), json.RawMessage(`[]`))
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
