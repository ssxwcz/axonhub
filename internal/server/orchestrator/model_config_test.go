package orchestrator

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/server/biz"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/transformer"
)

func TestApplyChannelModelConfig(t *testing.T) {
	effort := "high"
	budget := int64(8000)
	summary := "auto"

	t.Run("nil config returns request unchanged", func(t *testing.T) {
		req := &llm.Request{Model: "m"}
		require.Same(t, req, applyChannelModelConfig(req, nil))
	})

	t.Run("config without reasoning returns request unchanged", func(t *testing.T) {
		req := &llm.Request{Model: "m"}
		cfg := &objects.ChannelModelConfig{Model: "m", APIFormat: llm.APIFormatAnthropicMessage.String()}
		require.Same(t, req, applyChannelModelConfig(req, cfg))
	})

	t.Run("force disable clears reasoning fields", func(t *testing.T) {
		req := &llm.Request{
			Model:            "m",
			ReasoningEffort:  effort,
			ReasoningBudget:  &budget,
			ReasoningSummary: &summary,
		}
		cfg := &objects.ChannelModelConfig{
			Model:     "m",
			Reasoning: &objects.ModelReasoningConfig{Enabled: lo.ToPtr(false)},
		}

		got := applyChannelModelConfig(req, cfg)

		require.NotSame(t, req, got)
		require.Empty(t, got.ReasoningEffort)
		require.Nil(t, got.ReasoningBudget)
		require.Nil(t, got.ReasoningSummary)
		// The original request is not mutated.
		require.Equal(t, effort, req.ReasoningEffort)
		require.NotNil(t, req.ReasoningBudget)
	})

	t.Run("force disable without reasoning fields returns request unchanged", func(t *testing.T) {
		req := &llm.Request{Model: "m"}
		cfg := &objects.ChannelModelConfig{
			Model:     "m",
			Reasoning: &objects.ModelReasoningConfig{Enabled: lo.ToPtr(false)},
		}

		require.Same(t, req, applyChannelModelConfig(req, cfg))
	})

	t.Run("enabled nil keeps request values", func(t *testing.T) {
		req := &llm.Request{Model: "m", ReasoningEffort: effort}
		cfg := &objects.ChannelModelConfig{
			Model:     "m",
			Reasoning: &objects.ModelReasoningConfig{DefaultEffort: "low"},
		}

		require.Same(t, req, applyChannelModelConfig(req, cfg))
	})

	t.Run("defaults fill empty request", func(t *testing.T) {
		req := &llm.Request{Model: "m"}
		cfg := &objects.ChannelModelConfig{
			Model: "m",
			Reasoning: &objects.ModelReasoningConfig{
				DefaultEffort: effort,
				DefaultBudget: &budget,
			},
		}

		got := applyChannelModelConfig(req, cfg)

		require.NotSame(t, req, got)
		require.Equal(t, effort, got.ReasoningEffort)
		require.NotNil(t, got.ReasoningBudget)
		require.Equal(t, budget, *got.ReasoningBudget)
	})

	t.Run("defaults do not fill request with explicit effort", func(t *testing.T) {
		req := &llm.Request{Model: "m", ReasoningEffort: "low"}
		cfg := &objects.ChannelModelConfig{
			Model: "m",
			Reasoning: &objects.ModelReasoningConfig{
				DefaultEffort: effort,
				DefaultBudget: &budget,
			},
		}

		require.Same(t, req, applyChannelModelConfig(req, cfg))
	})

	t.Run("defaults do not fill request with explicit budget", func(t *testing.T) {
		req := &llm.Request{Model: "m", ReasoningBudget: &budget}
		cfg := &objects.ChannelModelConfig{
			Model:     "m",
			Reasoning: &objects.ModelReasoningConfig{DefaultEffort: effort},
		}

		require.Same(t, req, applyChannelModelConfig(req, cfg))
	})

	t.Run("force disable wins over defaults", func(t *testing.T) {
		req := &llm.Request{Model: "m"}
		cfg := &objects.ChannelModelConfig{
			Model: "m",
			Reasoning: &objects.ModelReasoningConfig{
				Enabled:       lo.ToPtr(false),
				DefaultEffort: effort,
			},
		}

		got := applyChannelModelConfig(req, cfg)

		require.Empty(t, got.ReasoningEffort)
		require.Nil(t, got.ReasoningBudget)
	})

	t.Run("effort map rewrites request effort", func(t *testing.T) {
		req := &llm.Request{Model: "m", ReasoningEffort: "max"}
		mapped := "xhigh"
		cfg := &objects.ChannelModelConfig{
			Model: "m",
			Reasoning: &objects.ModelReasoningConfig{
				EffortMap: map[string]*string{"max": &mapped},
			},
		}

		got := applyChannelModelConfig(req, cfg)

		require.NotSame(t, req, got)
		require.Equal(t, "xhigh", got.ReasoningEffort)
		// The original request is not mutated.
		require.Equal(t, "max", req.ReasoningEffort)
	})

	t.Run("effort map null clears reasoning", func(t *testing.T) {
		req := &llm.Request{Model: "m", ReasoningEffort: "high", ReasoningBudget: &budget}
		cfg := &objects.ChannelModelConfig{
			Model: "m",
			Reasoning: &objects.ModelReasoningConfig{
				EffortMap: map[string]*string{"high": nil},
			},
		}

		got := applyChannelModelConfig(req, cfg)

		require.NotSame(t, req, got)
		require.Empty(t, got.ReasoningEffort)
		require.Nil(t, got.ReasoningBudget)
	})

	t.Run("effort map applies to default effort", func(t *testing.T) {
		req := &llm.Request{Model: "m"}
		mapped := "medium"
		cfg := &objects.ChannelModelConfig{
			Model: "m",
			Reasoning: &objects.ModelReasoningConfig{
				DefaultEffort: "high",
				EffortMap:     map[string]*string{"high": &mapped},
			},
		}

		got := applyChannelModelConfig(req, cfg)

		require.NotSame(t, req, got)
		require.Equal(t, "medium", got.ReasoningEffort)
	})

	t.Run("effort map missing key passes through unchanged", func(t *testing.T) {
		req := &llm.Request{Model: "m", ReasoningEffort: "low"}
		mapped := "xhigh"
		cfg := &objects.ChannelModelConfig{
			Model: "m",
			Reasoning: &objects.ModelReasoningConfig{
				EffortMap: map[string]*string{"max": &mapped},
			},
		}

		require.Same(t, req, applyChannelModelConfig(req, cfg))
	})
}

func TestSelectModelConfigOutbound(t *testing.T) {
	primaryOutbound := &mockTransformer{apiFormat: llm.APIFormatOpenAIChatCompletion}
	anthropicOutbound := &mockTransformer{apiFormat: llm.APIFormatAnthropicMessage}

	chatFormat := llm.APIFormatOpenAIChatCompletion.String()
	anthropicFormat := llm.APIFormatAnthropicMessage.String()

	newCandidate := func(settings *objects.ChannelSettings) *ChannelModelsCandidate {
		return &ChannelModelsCandidate{
			Channel: &biz.Channel{
				Channel: &ent.Channel{ID: 1, Name: "test", Settings: settings},
				Outbound: primaryOutbound,
				Outbounds: map[string]transformer.Outbound{
					chatFormat: primaryOutbound,
				},
				ModelConfigOutbounds: map[string]transformer.Outbound{
					biz.ModelConfigOutboundKey(anthropicFormat, ""):       anthropicOutbound,
					biz.ModelConfigOutboundKey(chatFormat, "/v2/custom"):  primaryOutbound,
					biz.ModelConfigOutboundKey(anthropicFormat, "/anth"):  anthropicOutbound,
				},
			},
			APIFormat: chatFormat,
			Models:    []biz.ChannelModelEntry{{RequestModel: "m", ActualModel: "m"}},
		}
	}

	chatReq := &llm.Request{Model: "m", RequestType: llm.RequestTypeChat}

	t.Run("no config falls back", func(t *testing.T) {
		candidate := newCandidate(nil)
		_, ok := selectModelConfigOutbound(candidate, candidate.Models[0], chatReq)
		require.False(t, ok)
	})

	t.Run("reasoning-only config falls back", func(t *testing.T) {
		candidate := newCandidate(&objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Reasoning: &objects.ModelReasoningConfig{DefaultEffort: "high"}},
			},
		})
		_, ok := selectModelConfigOutbound(candidate, candidate.Models[0], chatReq)
		require.False(t, ok)
	})

	t.Run("api format override uses channel endpoint outbound when present", func(t *testing.T) {
		candidate := newCandidate(&objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", APIFormat: chatFormat},
			},
		})

		out, ok := selectModelConfigOutbound(candidate, candidate.Models[0], chatReq)
		require.True(t, ok)
		require.Same(t, primaryOutbound, out)
	})

	t.Run("api format override without endpoint uses prebuilt default-path outbound", func(t *testing.T) {
		candidate := newCandidate(&objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", APIFormat: anthropicFormat},
			},
		})

		out, ok := selectModelConfigOutbound(candidate, candidate.Models[0], chatReq)
		require.True(t, ok)
		require.Same(t, anthropicOutbound, out)
	})

	t.Run("api format and path override uses prebuilt outbound", func(t *testing.T) {
		candidate := newCandidate(&objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", APIFormat: anthropicFormat, Path: "/anth"},
			},
		})

		out, ok := selectModelConfigOutbound(candidate, candidate.Models[0], chatReq)
		require.True(t, ok)
		require.Same(t, anthropicOutbound, out)
	})

	t.Run("path-only override follows selected format", func(t *testing.T) {
		candidate := newCandidate(&objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Path: "/v2/custom"},
			},
		})

		out, ok := selectModelConfigOutbound(candidate, candidate.Models[0], chatReq)
		require.True(t, ok)
		require.Same(t, primaryOutbound, out)
	})

	t.Run("override skipped when format cannot serve request type", func(t *testing.T) {
		candidate := newCandidate(&objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", APIFormat: anthropicFormat},
			},
		})

		embeddingReq := &llm.Request{Model: "m", RequestType: llm.RequestTypeEmbedding}

		_, ok := selectModelConfigOutbound(candidate, candidate.Models[0], embeddingReq)
		require.False(t, ok)
	})

	t.Run("config matches actual model not request model", func(t *testing.T) {
		candidate := newCandidate(&objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "other-model", APIFormat: anthropicFormat},
			},
		})

		_, ok := selectModelConfigOutbound(candidate, candidate.Models[0], chatReq)
		require.False(t, ok)
	})
}
