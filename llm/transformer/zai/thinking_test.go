package zai

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/llm"
)

func TestSupportsNativeReasoningEffort(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"glm-4.5", false},
		{"glm-4.7", false},
		{"glm-4.7-flash", false},
		{"glm-5.0", false},
		{"glm-5.1", false},
		{"glm-5.2", true},
		{"glm-5.3", true},
		{"glm-5.3-flash", true},
		{"zai-org/glm-5.3:thinking", true},
		{"GLM-5.3-FLASH", true},
		{"gpt-4", false},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			assert.Equal(t, tt.want, supportsNativeReasoningEffort(tt.model))
		})
	}
}

func TestIsAlwaysThinkingModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"glm-5.3", true},
		{"glm-5.3-flash", true},
		{"zai-org/glm-5.3:thinking", true},
		{"GLM-5.3-FLASH", true},
		{"glm-5.2", false},
		{"glm-4.7", false},
		{"gpt-4", false},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			assert.Equal(t, tt.want, isAlwaysThinkingModel(tt.model))
		})
	}
}

func TestNormalizeGLM53Effort(t *testing.T) {
	tests := []struct {
		effort string
		want   string
	}{
		{"none", "low"},
		{"minimal", "low"},
		{"low", "low"},
		{"medium", "high"},
		{"high", "high"},
		{"xhigh", "max"},
		{"max", "max"},
		{"unknown", "low"},
	}
	for _, tt := range tests {
		t.Run(tt.effort, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeGLM53Effort(tt.effort))
		})
	}
}

func TestResolveZaiThinking(t *testing.T) {
	tests := []struct {
		name            string
		model           string
		reasoningEffort string
		mappings        []llm.ReasoningEffortMapping
		wantThinking    *Thinking
		wantEffort      *string
	}{
		{
			name:            "glm-5.3 none maps to low with thinking enabled",
			model:           "glm-5.3-flash",
			reasoningEffort: "none",
			wantThinking:    &Thinking{Type: "enabled"},
			wantEffort:      lo.ToPtr("low"),
		},
		{
			name:            "glm-5.3 high passes through",
			model:           "glm-5.3",
			reasoningEffort: "high",
			wantThinking:    &Thinking{Type: "enabled"},
			wantEffort:      lo.ToPtr("high"),
		},
		{
			name:            "glm-5.3 xhigh maps to max",
			model:           "glm-5.3",
			reasoningEffort: "xhigh",
			wantThinking:    &Thinking{Type: "enabled"},
			wantEffort:      lo.ToPtr("max"),
		},
		{
			name:            "glm-5.3 channel mapping none to low",
			model:           "glm-5.3",
			reasoningEffort: "none",
			mappings:        []llm.ReasoningEffortMapping{{From: "none", To: "low"}},
			wantThinking:    &Thinking{Type: "enabled"},
			wantEffort:      lo.ToPtr("low"),
		},
		{
			name:            "glm-5.3 channel mapping xhigh to max",
			model:           "glm-5.3",
			reasoningEffort: "xhigh",
			mappings:        []llm.ReasoningEffortMapping{{From: "xhigh", To: "max"}},
			wantThinking:    &Thinking{Type: "enabled"},
			wantEffort:      lo.ToPtr("max"),
		},
		{
			name:            "glm-5.2 none disables thinking",
			model:           "glm-5.2",
			reasoningEffort: "none",
			wantThinking:    &Thinking{Type: "disabled"},
			wantEffort:      nil,
		},
		{
			name:            "glm-5.2 low passes reasoning_effort",
			model:           "glm-5.2",
			reasoningEffort: "low",
			wantThinking:    &Thinking{Type: "enabled"},
			wantEffort:      lo.ToPtr("low"),
		},
		{
			name:            "glm-4.7 none disables thinking",
			model:           "glm-4.7",
			reasoningEffort: "none",
			wantThinking:    &Thinking{Type: "disabled"},
			wantEffort:      nil,
		},
		{
			name:            "glm-4.7 high enables thinking",
			model:           "glm-4.7",
			reasoningEffort: "high",
			wantThinking:    &Thinking{Type: "enabled"},
			wantEffort:      nil,
		},
		{
			name:            "empty effort returns nil thinking",
			model:           "glm-5.3",
			reasoningEffort: "",
			wantThinking:    nil,
			wantEffort:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			thinking, effort := resolveZaiThinking(tt.model, tt.reasoningEffort, tt.mappings)
			if tt.wantThinking == nil {
				assert.Nil(t, thinking)
			} else {
				require.NotNil(t, thinking)
				assert.Equal(t, tt.wantThinking.Type, thinking.Type)
			}
			if tt.wantEffort == nil {
				assert.Nil(t, effort)
			} else {
				require.NotNil(t, effort)
				assert.Equal(t, *tt.wantEffort, *effort)
			}
		})
	}
}
