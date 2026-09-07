package opencode

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/pipeline"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/shared"
)

// sessionHeaderOutbound adds the OpenCode Go conversation session header to a
// protocol-specific outbound transformer. It decorates the single-protocol
// transformers used by the dedicated opencode_go, opencode_go_anthropic and
// opencode_go_responses channels; this fork does not use the upstream
// model-routing opencode transformer.
type sessionHeaderOutbound struct {
	transformer.Outbound
}

// sessionHeaderCustomizedOutbound preserves customized executor behavior for
// outbounds such as OpenAI Responses while adding OpenCode session affinity.
type sessionHeaderCustomizedOutbound struct {
	*sessionHeaderOutbound

	customizer pipeline.ChannelCustomizedExecutor
}

// WithSessionHeader decorates an outbound transformer with OpenCode Go session
// affinity so every outbound inference request carries the stable
// per-conversation x-opencode-session header OpenCode Go requires.
func WithSessionHeader(outbound transformer.Outbound) transformer.Outbound {
	if outbound == nil {
		return nil
	}

	wrapped := &sessionHeaderOutbound{Outbound: outbound}
	if customizer, ok := outbound.(pipeline.ChannelCustomizedExecutor); ok {
		return &sessionHeaderCustomizedOutbound{
			sessionHeaderOutbound: wrapped,
			customizer:            customizer,
		}
	}

	return wrapped
}

func (t *sessionHeaderOutbound) TransformRequest(ctx context.Context, llmReq *llm.Request) (*httpclient.Request, error) {
	httpReq, err := t.Outbound.TransformRequest(ctx, llmReq)
	if err != nil {
		return nil, err
	}

	setSessionHeader(ctx, llmReq, httpReq)

	return httpReq, nil
}

// CustomizeExecutor forwards executor customization to the wrapped outbound.
func (t *sessionHeaderCustomizedOutbound) CustomizeExecutor(executor pipeline.Executor) pipeline.Executor {
	return t.customizer.CustomizeExecutor(executor)
}

// FinalizeTransportRequest forwards transport cleanup when supported.
func (t *sessionHeaderCustomizedOutbound) FinalizeTransportRequest(request *httpclient.Request) *httpclient.Request {
	if finalizer, ok := t.Outbound.(transformer.TransportRequestFinalizer); ok {
		return finalizer.FinalizeTransportRequest(request)
	}

	return request
}

// Stop releases resources owned by the wrapped outbound when supported.
func (t *sessionHeaderCustomizedOutbound) Stop() {
	if stoppable, ok := t.Outbound.(interface{ Stop() }); ok {
		stoppable.Stop()
	}
}

func setSessionHeader(ctx context.Context, llmReq *llm.Request, httpReq *httpclient.Request) {
	if httpReq == nil {
		return
	}

	if httpReq.Headers == nil {
		httpReq.Headers = make(http.Header)
	}

	httpReq.Headers.Set(SessionHeader, resolveSessionID(ctx, llmReq))
}

// resolveSessionID returns the OpenCode Go session identifier for an outbound
// request. It prefers the stable per-conversation id the client sent on the
// inbound headers, then the id the trace middleware recorded in the context
// (which reuses those headers or provider turn/user metadata), and only falls
// back to a fresh UUID when the inbound request carried no conversation
// identity at all.
func resolveSessionID(ctx context.Context, llmReq *llm.Request) string {
	if llmReq != nil && llmReq.RawRequest != nil {
		if sessionID := GetSessionIDFromHeaders(llmReq.RawRequest.Headers); sessionID != "" {
			return sessionID
		}
	}

	if sessionID, ok := shared.GetSessionID(ctx); ok {
		if sessionID = strings.TrimSpace(sessionID); sessionID != "" {
			return sessionID
		}
	}

	return uuid.NewString()
}
