package config

import "testing"

func TestParseDependency_SimpleStageID(t *testing.T) {
	ref := ParseDependency("my_stage")

	if ref.StageID != "my_stage" {
		t.Errorf("Expected StageID 'my_stage', got '%s'", ref.StageID)
	}

	if ref.Branch != "" {
		t.Errorf("Expected empty Branch, got '%s'", ref.Branch)
	}
}

func TestParseDependency_WithBranch(t *testing.T) {
	tests := []struct {
		input          string
		expectedID     string
		expectedBranch string
	}{
		{"check:true", "check", "true"},
		{"check:false", "check", "false"},
		{"my_if_step:success", "my_if_step", "success"},
		{"step:branch_name", "step", "branch_name"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			ref := ParseDependency(tc.input)

			if ref.StageID != tc.expectedID {
				t.Errorf("Expected StageID '%s', got '%s'", tc.expectedID, ref.StageID)
			}

			if ref.Branch != tc.expectedBranch {
				t.Errorf("Expected Branch '%s', got '%s'", tc.expectedBranch, ref.Branch)
			}
		})
	}
}

func TestParseDependency_EmptyBranch(t *testing.T) {
	// Trailing colon should be treated as no branch
	ref := ParseDependency("stage:")

	if ref.StageID != "stage:" {
		t.Errorf("Expected StageID 'stage:', got '%s'", ref.StageID)
	}

	if ref.Branch != "" {
		t.Errorf("Expected empty Branch for trailing colon, got '%s'", ref.Branch)
	}
}

func TestParseDependency_MultipleColons(t *testing.T) {
	// Multiple colons: use the last one as separator
	// "stage:with:colons:branch" -> StageID="stage:with:colons", Branch="branch"
	ref := ParseDependency("stage:with:colons:branch")

	if ref.StageID != "stage:with:colons" {
		t.Errorf("Expected StageID 'stage:with:colons', got '%s'", ref.StageID)
	}

	if ref.Branch != "branch" {
		t.Errorf("Expected Branch 'branch', got '%s'", ref.Branch)
	}
}
