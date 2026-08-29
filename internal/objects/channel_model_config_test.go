package objects

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestChannelSettingsGetModelConfig(t *testing.T) {
	t.Run("nil settings returns nil", func(t *testing.T) {
		var settings *ChannelSettings
		require.Nil(t, settings.GetModelConfig("m"))
	})

	t.Run("no model configs returns nil", func(t *testing.T) {
		settings := &ChannelSettings{}
		require.Nil(t, settings.GetModelConfig("m"))
	})

	t.Run("empty model name returns nil", func(t *testing.T) {
		settings := &ChannelSettings{
			ModelConfigs: []ChannelModelConfig{{Model: "m"}},
		}
		require.Nil(t, settings.GetModelConfig(""))
	})

	t.Run("returns config by actual model name", func(t *testing.T) {
		settings := &ChannelSettings{
			ModelConfigs: []ChannelModelConfig{
				{Model: "other"},
				{Model: "m", APIFormat: "anthropic/messages"},
			},
		}

		cfg := settings.GetModelConfig("m")
		require.NotNil(t, cfg)
		require.Equal(t, "anthropic/messages", cfg.APIFormat)
		require.Nil(t, settings.GetModelConfig("missing"))
	})
}

func TestChannelModelConfigIsEmpty(t *testing.T) {
	require.True(t, (*ChannelModelConfig)(nil).IsEmpty())
	require.True(t, (&ChannelModelConfig{Model: "m"}).IsEmpty())
	require.True(t, (&ChannelModelConfig{Model: "m", Reasoning: &ModelReasoningConfig{}}).IsEmpty())
	require.False(t, (&ChannelModelConfig{Model: "m", APIFormat: "openai/chat_completions"}).IsEmpty())
	require.False(t, (&ChannelModelConfig{Model: "m", Path: "/x"}).IsEmpty())
	require.False(t, (&ChannelModelConfig{Model: "m", Reasoning: &ModelReasoningConfig{DefaultEffort: "high"}}).IsEmpty())
	require.False(t, (&ChannelModelConfig{Model: "m", Reasoning: &ModelReasoningConfig{Enabled: lo.ToPtr(false)}}).IsEmpty())
}

func TestModelReasoningConfigIsEmpty(t *testing.T) {
	require.True(t, (*ModelReasoningConfig)(nil).IsEmpty())
	require.True(t, (&ModelReasoningConfig{}).IsEmpty())
	require.False(t, (&ModelReasoningConfig{DefaultBudget: lo.ToPtr(int64(1))}).IsEmpty())
}
