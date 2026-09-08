package space

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	if err := decodeFixture(data, &suite); err != nil {
		t.Fatal(err)
	}
	if suite.SchemaVersion != 1 {
		t.Fatalf("schema version = %d", suite.SchemaVersion)
	}
	for _, tc := range suite.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			state := tc.Initial
			for i, command := range tc.Commands {
				transition := Apply(state, command)
				want := tc.Expected.Transitions[i]
				if transition.Receipt != want.Receipt || transition.Rejection != want.Rejection {
					t.Fatalf("transition mismatch: got %#v want %#v", transition, want)
				}
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

func TestFixtureDecoderRejectsUnknownEnums(t *testing.T) {
	for name, data := range map[string]string{
		"grant":         `{"schemaVersion":1,"cases":[{"name":"x","initial":{"schemaVersion":1,"audit":[],"capabilities":[{"id":"x","grant":"maybe"}]},"commands":[],"expected":{"state":{"schemaVersion":1,"audit":[],"transitions":[]},"audit":[],"transitions":[]}}]}`,
		"identity kind": `{"schemaVersion":1,"cases":[{"name":"x","initial":{"schemaVersion":1,"audit":[],"identities":[{"id":"x","kind":"maybe"}]},"commands":[],"expected":{"state":{"schemaVersion":1,"audit":[],"transitions":[]},"audit":[],"transitions":[]}}]}`,
		"audit event":   `{"schemaVersion":1,"cases":[{"name":"x","initial":{"schemaVersion":1,"audit":[{"event":"maybe","requestId":"x"}]},"commands":[],"expected":{"state":{"schemaVersion":1,"audit":[],"transitions":[]},"audit":[],"transitions":[]}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			var suite FixtureSuite
			if err := decodeFixture([]byte(data), &suite); err == nil {
				t.Fatal("expected unknown enum error")
			}
		})
	}
}

func decodeFixture(data []byte, suite *FixtureSuite) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(suite); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing JSON: %v", err)
	}
	if suite.SchemaVersion != 1 {
		return fmt.Errorf("unsupported fixture schema version: %d", suite.SchemaVersion)
	}
	for _, tc := range suite.Cases {
		for _, identity := range tc.Initial.Identities {
			if identity.Kind != IdentityOwner && identity.Kind != IdentityHost {
				return fmt.Errorf("unknown identity kind: %q", identity.Kind)
			}
		}
		for _, capability := range tc.Initial.Capabilities {
			if capability.Grant != GrantDeny && capability.Grant != GrantAsk && capability.Grant != GrantAllow {
				return fmt.Errorf("unknown grant: %q", capability.Grant)
			}
		}
		for _, event := range tc.Initial.Audit {
			if event.Event != AuditSpaceCreated {
				return fmt.Errorf("unknown audit event: %q", event.Event)
			}
		}
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
