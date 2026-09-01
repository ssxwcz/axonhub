package orchestrator

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/llm/httpclient"
)

func TestApplyUpstreamRequestID_ReusedClientIDGetsUniqueUpstreamIDs(t *testing.T) {
	middleware := applyUpstreamRequestID()

	firstReq := &httpclient.Request{Headers: http.Header{}}
	firstReq.Headers.Set(upstreamRequestIDHeader, "shared-agent-trace")
	first, err := middleware.OnOutboundRawRequest(
		contexts.WithRequestID(t.Context(), "ar-first-request"),
		firstReq,
	)
	require.NoError(t, err)

	secondReq := &httpclient.Request{Headers: http.Header{}}
	secondReq.Headers.Set(upstreamRequestIDHeader, "shared-agent-trace")
	second, err := middleware.OnOutboundRawRequest(
		contexts.WithRequestID(t.Context(), "ar-second-request"),
		secondReq,
	)
	require.NoError(t, err)

	assert.Equal(t, "ar-first-request", first.Headers.Get(upstreamRequestIDHeader))
	assert.Equal(t, "ar-second-request", second.Headers.Get(upstreamRequestIDHeader))
	assert.NotEqual(t, first.Headers.Get(upstreamRequestIDHeader), second.Headers.Get(upstreamRequestIDHeader))
}

func TestApplyUpstreamRequestID_InitializesHeaders(t *testing.T) {
	ctx := contexts.WithRequestID(t.Context(), "ar-unique-request")

	got, err := applyUpstreamRequestID().OnOutboundRawRequest(ctx, &httpclient.Request{})
	require.NoError(t, err)
	assert.Equal(t, "ar-unique-request", got.Headers.Get(upstreamRequestIDHeader))
}

func TestApplyUpstreamRequestID_PreservesHeaderWithoutContextID(t *testing.T) {
	request := &httpclient.Request{Headers: http.Header{}}
	request.Headers.Set(upstreamRequestIDHeader, "client-request-id")

	got, err := applyUpstreamRequestID().OnOutboundRawRequest(t.Context(), request)
	require.NoError(t, err)
	assert.Equal(t, "client-request-id", got.Headers.Get(upstreamRequestIDHeader))
}
