package opencode

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/shared"
)

type stubOutbound struct {
	transformer.Outbound
}

func (s stubOutbound) TransformRequest(context.Context, *llm.Request) (*httpclient.Request, error) {
	return &httpclient.Request{Headers: http.Header{}}, nil
}

func TestWithSessionHeader_AddsHeader(t *testing.T) {
	tr := WithSessionHeader(stubOutbound{})
	ctx := shared.WithSessionID(context.Background(), "ctx-session")

	httpReq, err := tr.TransformRequest(ctx, &llm.Request{
		Model:    "glm-5.2",
		Messages: []llm.Message{{Role: "user", Content: llm.MessageContent{Content: lo.ToPtr("Hello")}}},
	})
	require.NoError(t, err)
	assert.Equal(t, "ctx-session", httpReq.Headers.Get(SessionHeader))
}

func TestResolveSessionID_PrefersInboundHeadersOverContext(t *testing.T) {
	tr := WithSessionHeader(stubOutbound{})
	ctx := shared.WithSessionID(context.Background(), "ctx-session")

	httpReq, err := tr.TransformRequest(ctx, &llm.Request{
		Model:      "glm-5.2",
		Messages:   []llm.Message{{Role: "user", Content: llm.MessageContent{Content: lo.ToPtr("Hello")}}},
		RawRequest: &httpclient.Request{Headers: http.Header{SessionHeader: {"client-session"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, "client-session", httpReq.Headers.Get(SessionHeader))
}

func TestResolveSessionID_FallsBackToUUID(t *testing.T) {
	tr := WithSessionHeader(stubOutbound{})

	httpReq, err := tr.TransformRequest(context.Background(), &llm.Request{
		Model:    "glm-5.2",
		Messages: []llm.Message{{Role: "user", Content: llm.MessageContent{Content: lo.ToPtr("Hello")}}},
	})
	require.NoError(t, err)
	sessionID := httpReq.Headers.Get(SessionHeader)
	require.NotEmpty(t, sessionID)
	_, err = uuid.Parse(sessionID)
	assert.NoError(t, err, "fallback session id must be a valid UUID")
}

func TestWithSessionHeader_NilOutbound(t *testing.T) {
	assert.Nil(t, WithSessionHeader(nil))
}
