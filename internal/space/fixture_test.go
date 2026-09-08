package space

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFixtures(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "protocol", "fixtures", "space-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var suite FixtureSuite
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&suite); err != nil {
		t.Fatal(err)
	}
	if suite.SchemaVersion != 1 {
		t.Fatalf("schema version = %d", suite.SchemaVersion)
	}
	for _, tc := range suite.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			state := tc.Initial
			for _, command := range tc.Commands {
				transition := Apply(state, command)
				if transition.Rejection != "" {
					t.Fatalf("unexpected rejection: %s", transition.Rejection)
				}
				state = transition.State
			}
			if !equalJSON(state, tc.Expected.State) {
				t.Fatalf("state mismatch: got %#v want %#v", state, tc.Expected.State)
			}
			if !equalJSONSlice(state.Audit, tc.Expected.Audit) {
				t.Fatalf("audit mismatch: got %#v want %#v", state.Audit, tc.Expected.Audit)
			}
		})
	}
}

func TestFixtureDecoderRejectsUnsupportedSchema(t *testing.T) {
	var suite FixtureSuite
	err := decodeFixture([]byte(`{"schemaVersion":2,"cases":[]}`), &suite)
	if err == nil {
		t.Fatal("expected unsupported schema error")
	}
}

func TestFixtureDecoderRejectsUnknownField(t *testing.T) {
	var suite FixtureSuite
	err := decodeFixture([]byte(`{"schemaVersion":1,"cases":[],"unexpected":true}`), &suite)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
}

func decodeFixture(data []byte, suite *FixtureSuite) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(suite); err != nil {
		return err
	}
	if suite.SchemaVersion != 1 {
		return fmt.Errorf("unsupported fixture schema version: %d", suite.SchemaVersion)
	}
	return nil
}

func equalJSON(a, b State) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func equalJSONSlice(a, b []AuditEvent) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
