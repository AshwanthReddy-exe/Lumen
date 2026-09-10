package setup

import "testing"

func TestOutcomeValuesAreStable(t *testing.T) {
	for _, got := range []Outcome{Ready, Degraded, ActionRequired} {
		if err := got.Validate(); err != nil {
			t.Fatalf("%q: %v", got, err)
		}
	}
}
