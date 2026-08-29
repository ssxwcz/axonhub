package anthropic

func supportsAdaptiveThinking(config *Config) bool {
	if config == nil {
		return true
	}

	//nolint:exhaustive // Checked.
	switch config.Type {
	case PlatformDirect, PlatformClaudeCode, PlatformBedrock, PlatformVertex:
		return true
	default:
		return false
	}
}

// supportsOutputConfig returns true if the platform supports the output_config field
// with effort control. DeepSeek supports output_config.effort but does NOT support
// thinking.type = "adaptive".
func supportsOutputConfig(config *Config) bool {
	if config == nil {
		return true
	}

	//nolint:exhaustive // Checked.
	switch config.Type {
	case PlatformDirect, PlatformClaudeCode, PlatformBedrock, PlatformVertex, PlatformDeepSeek:
		return true
	default:
		return false
	}
}

// thinkingBudgetToReasoningEffort converts thinking budget tokens to a reasoning
// effort string.
//
// The buckets are the inverse of defaultReasoningEffortMapping, so an effort that
// is converted to a budget and back resolves to the same level. The high and max
// boundaries follow what clients actually put on the wire: Anthropic-protocol
// clients (for example opencode through the AI SDK) encode "high" as ~16000
// thinking tokens and "max" as ~32000. Bucketing everything above 20000 as "high"
// silently downgraded those clients' "max" to "high".
func thinkingBudgetToReasoningEffort(budgetTokens int64) string {
	switch {
	case budgetTokens <= 5000:
		return "low"
	case budgetTokens <= 15000:
		return "medium"
	case budgetTokens <= 20000:
		return "high"
	case budgetTokens <= 31000:
		return "xhigh"
	default:
		return "max"
	}
}

// getDefaultReasoningEffortMapping returns the default mapping from ReasoningEffort to thinking budget tokens.
//
// Each level maps to a distinct budget so that a higher effort really thinks
// longer. Values must stay inside the matching bucket of
// thinkingBudgetToReasoningEffort, otherwise a round trip through the wire would
// downgrade the effort level.
var defaultReasoningEffortMapping = map[string]int64{
	"low":    5000,
	"medium": 15000,
	"high":   16000,
	"xhigh":  24000,
	"max":    31999,
}

// getThinkingBudgetTokensWithConfig returns the thinking budget tokens for a given reasoning effort with config.
func getThinkingBudgetTokensWithConfig(reasoningEffort string, config *Config) int64 {
	if config != nil && config.ReasoningEffortToBudget != nil {
		if budget, exists := config.ReasoningEffortToBudget[reasoningEffort]; exists {
			return budget
		}
	}

	// Fall back to default mapping
	if budget, exists := defaultReasoningEffortMapping[reasoningEffort]; exists {
		return budget
	}

	// Default to medium if not found
	return 15000
}
