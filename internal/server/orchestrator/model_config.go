package orchestrator

import (
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/server/biz"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/transformer"
)

// candidateFirstEntry returns the candidate's first model entry, or an empty
// entry when the candidate carries no models.
func candidateFirstEntry(candidate *ChannelModelsCandidate) biz.ChannelModelEntry {
	if candidate == nil || len(candidate.Models) == 0 {
		return biz.ChannelModelEntry{}
	}

	return candidate.Models[0]
}

// channelModelConfigForEntry returns the per-model configuration attached to
// the candidate's channel for the entry's actual model, or nil when the model
// has no configuration. Matching happens on ActualModel (the name sent to the
// provider) so every request alias of one upstream model shares the config.
func channelModelConfigForEntry(candidate *ChannelModelsCandidate, entry biz.ChannelModelEntry) *objects.ChannelModelConfig {
	if candidate == nil || candidate.Channel == nil {
		return nil
	}

	return candidate.Channel.Settings.GetModelConfig(entry.ActualModel)
}

// modelConfigHasEndpointOverride reports whether the config overrides the
// outbound api format or the endpoint path.
func modelConfigHasEndpointOverride(cfg *objects.ChannelModelConfig) bool {
	return cfg != nil && (cfg.APIFormat != "" || cfg.Path != "")
}

// selectModelConfigOutbound resolves the outbound transformer for a
// model-level endpoint override. The second return value reports whether an
// override transformer was found; when false the caller falls back to the
// regular channel endpoint selection.
//
// Resolution rules:
//   - APIFormat set, Path empty: the channel's endpoint outbound for that
//     format wins when present (including its path); otherwise the
//     default-path outbound pre-built for the model config is used.
//   - APIFormat set, Path set: the outbound bound to that exact path.
//   - APIFormat empty, Path set: the path applies to the format the channel
//     would otherwise select for the request.
//
// The override is skipped when the request type cannot be served by the
// configured format, so e.g. a chat-only override never hijacks an embedding
// request.
func selectModelConfigOutbound(
	candidate *ChannelModelsCandidate,
	entry biz.ChannelModelEntry,
	req *llm.Request,
) (transformer.Outbound, bool) {
	if candidate == nil || candidate.Channel == nil {
		return nil, false
	}

	cfg := channelModelConfigForEntry(candidate, entry)
	if !modelConfigHasEndpointOverride(cfg) {
		return nil, false
	}

	format := cfg.APIFormat
	if format == "" {
		format = candidate.APIFormat
	}
	if format == "" {
		return nil, false
	}

	if req != nil {
		if allowed := llm.CapableAPIFormats(req.RequestType); allowed != nil {
			if _, ok := allowed[format]; !ok {
				return nil, false
			}
		}
	}

	channel := candidate.Channel

	if cfg.Path == "" {
		if out, ok := channel.Outbounds[format]; ok {
			return out, true
		}
	}

	if out, ok := channel.ModelConfigOutbounds[biz.ModelConfigOutboundKey(format, cfg.Path)]; ok {
		return out, true
	}

	return nil, false
}

// applyChannelModelConfig applies a model's reasoning configuration to the
// unified request after channel transform options and before the outbound
// transformer runs:
//
//   - Force-disable (Enabled == false) clears the reasoning fields so no
//     reasoning parameter reaches the provider. Clearing them here also
//     neutralizes downstream effort rewrites (channel-level
//     reasoningEffortMapping and the Claude Code mapping), because those
//     only ever rewrite a non-empty effort value.
//   - DefaultEffort/DefaultBudget only fill a request that carries no
//     reasoning intent at all (no effort, no budget), so explicit request
//     values — including efforts derived from the auto reasoning effort
//     model suffix — always win over the model defaults.
//   - EffortMap rewrites the final effort value (whether it came from the
//     request or the model default): a string target replaces the effort
//     sent upstream (rename/downgrade), a nil target marks the effort as
//     explicitly unsupported and clears reasoning for it. Missing keys
//     pass through unchanged.
func applyChannelModelConfig(req *llm.Request, cfg *objects.ChannelModelConfig) *llm.Request {
	if req == nil || cfg == nil || cfg.Reasoning == nil {
		return req
	}

	reasoning := cfg.Reasoning

	if reasoning.Enabled != nil && !*reasoning.Enabled {
		if req.ReasoningEffort == "" && req.ReasoningBudget == nil && req.ReasoningSummary == nil {
			return req
		}

		cloned := *req
		cloned.ReasoningEffort = ""
		cloned.ReasoningBudget = nil
		cloned.ReasoningSummary = nil

		return &cloned
	}

	// Fill model defaults only when the request carries no reasoning intent.
	if req.ReasoningEffort == "" && req.ReasoningBudget == nil && (reasoning.DefaultEffort != "" || reasoning.DefaultBudget != nil) {
		cloned := *req
		if reasoning.DefaultEffort != "" {
			cloned.ReasoningEffort = reasoning.DefaultEffort
		}
		if reasoning.DefaultBudget != nil {
			cloned.ReasoningBudget = reasoning.DefaultBudget
		}
		req = &cloned
	}

	// Apply the effort map to the final effort value (request or default).
	if req.ReasoningEffort != "" && len(reasoning.EffortMap) > 0 {
		if target, ok := reasoning.EffortMap[req.ReasoningEffort]; ok {
			cloned := *req
			if target == nil {
				cloned.ReasoningEffort = ""
				cloned.ReasoningBudget = nil
			} else {
				cloned.ReasoningEffort = *target
			}
			req = &cloned
		}
	}

	return req
}
