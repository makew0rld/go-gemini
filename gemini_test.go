package gemini

import "testing"

func TestSimplifyStatus(t *testing.T) {
	tests := []struct {
		ComplexStatus int
		SimpleStatus  int
	}{
		{10, 10},
		{20, 20},
		{21, 20},
		{44, 40},
		{59, 50},
	}

	for _, tt := range tests {
		result := SimplifyStatus(tt.ComplexStatus)
		if result != tt.SimpleStatus {
			t.Errorf("Expected the simplified status of %d to be %d, got %d instead", tt.ComplexStatus, tt.SimpleStatus, result)
		}
	}
}
