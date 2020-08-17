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

func TestQuery(t *testing.T) {
	query := `t/&^*% es\++\t`
	escaped := `t%2F&%5E%2A%25%20es%5C%2B%2B%5Ct`
	if QueryEscape(query) != escaped {
		t.Errorf("Query escape failed")
	}
	q, err := QueryUnescape(escaped)
	if q != query || err != nil {
		t.Errorf("Query unescape failed")
	}
}
