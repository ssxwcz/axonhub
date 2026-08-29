package biz

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/llm"
)

func TestNormalizeModelConfigs(t *testing.T) {
	t.Run("nil settings passes", func(t *testing.T) {
		require.NoError(t, NormalizeModelConfigs(nil))
	})

	t.Run("empty model rejected", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{{Model: "  "}},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "model is required")
	})

	t.Run("duplicate model rejected", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", APIFormat: llm.APIFormatAnthropicMessage.String()},
				{Model: "m", Path: "/x"},
			},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "duplicate model")
	})

	t.Run("unsupported api format rejected", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", APIFormat: "bogus/format"},
			},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "unsupported api_format")
	})

	t.Run("full URL path rejected", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Path: "https://example.com/v1"},
			},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "must not be a full URL")
	})

	t.Run("relative path rejected", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Path: "v1/messages"},
			},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "must start with '/'")
	})

	t.Run("invalid default effort rejected", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Reasoning: &objects.ModelReasoningConfig{DefaultEffort: "ultra"}},
			},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "unsupported default reasoning effort")
	})

	t.Run("non-positive budget rejected", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Reasoning: &objects.ModelReasoningConfig{DefaultBudget: lo.ToPtr(int64(0))}},
			},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "budget must be positive")
	})

	t.Run("valid effort map passes and trims values", func(t *testing.T) {
		target := " xhigh "
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Reasoning: &objects.ModelReasoningConfig{EffortMap: map[string]*string{"max": &target, "high": nil}}},
			},
		}

		require.NoError(t, NormalizeModelConfigs(settings))
		require.Len(t, settings.ModelConfigs, 1)
		mapped := settings.ModelConfigs[0].Reasoning.EffortMap["max"]
		require.NotNil(t, mapped)
		require.Equal(t, "xhigh", *mapped)
		require.Nil(t, settings.ModelConfigs[0].Reasoning.EffortMap["high"])
	})

	t.Run("unsupported effort map key rejected", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Reasoning: &objects.ModelReasoningConfig{EffortMap: map[string]*string{"ultra": nil}}},
			},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "unsupported effort map key")
	})

	t.Run("unsupported effort map value rejected", func(t *testing.T) {
		target := "ultra"
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Reasoning: &objects.ModelReasoningConfig{EffortMap: map[string]*string{"max": &target}}},
			},
		}
		require.ErrorContains(t, NormalizeModelConfigs(settings), "unsupported effort map value")
	})

	t.Run("empty entries dropped and values trimmed", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "no-op"},
				{Model: "  m  ", APIFormat: " " + llm.APIFormatAnthropicMessage.String() + " "},
			},
		}

		require.NoError(t, NormalizeModelConfigs(settings))
		require.Len(t, settings.ModelConfigs, 1)
		require.Equal(t, "m", settings.ModelConfigs[0].Model)
		require.Equal(t, llm.APIFormatAnthropicMessage.String(), settings.ModelConfigs[0].APIFormat)
	})

	t.Run("all empty entries collapse to nil", func(t *testing.T) {
		settings := &objects.ChannelSettings{
			ModelConfigs: []objects.ChannelModelConfig{
				{Model: "m", Reasoning: &objects.ModelReasoningConfig{Enabled: lo.ToPtr(true)}},
			},
		}

		require.NoError(t, NormalizeModelConfigs(settings))
		require.Nil(t, settings.ModelConfigs)
	})
}
