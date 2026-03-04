package llmagent

import (
	"context"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomID(t *testing.T) {
	tests := map[string]struct {
		prefix string
	}{
		"session prefix": {
			prefix: "sess",
		},
		"empty prefix": {
			prefix: "",
		},
		"custom prefix": {
			prefix: "test",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			id := randomID(tc.prefix)

			assert.True(t, strings.HasPrefix(id, tc.prefix+"_"), "expected prefix %q_, got %q", tc.prefix, id)

			// 12 random bytes = 24 hex chars + prefix + underscore
			expectedLen := len(tc.prefix) + 1 + 24
			assert.Len(t, id, expectedLen)
		})
	}
}

func TestRandomID_Uniqueness(t *testing.T) {
	ids := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		id := randomID("test")
		_, exists := ids[id]
		assert.False(t, exists, "duplicate ID generated: %s", id)
		ids[id] = struct{}{}
	}
}

func TestCleanupAllSessions(t *testing.T) {
	tests := map[string]struct {
		sessionCount int
	}{
		"no sessions": {
			sessionCount: 0,
		},
		"single session": {
			sessionCount: 1,
		},
		"multiple sessions": {
			sessionCount: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			agent := &acpAgent{
				sessions: make(map[acp.SessionId]*acpSession),
			}

			cancelledCount := 0
			for i := 0; i < tc.sessionCount; i++ {
				_, cancel := context.WithCancel(context.Background())
				sess := &acpSession{
					sessionCancel: func() {
						cancelledCount++
						cancel()
					},
				}
				agent.sessions[acp.SessionId(randomID("sess"))] = sess
			}

			require.Len(t, agent.sessions, tc.sessionCount)

			agent.cleanupAllSessions()

			assert.Empty(t, agent.sessions)
			assert.Equal(t, tc.sessionCount, cancelledCount)
		})
	}
}

func TestCleanupAllSessions_NoDoubleCleanupWithCancel(t *testing.T) {
	agent := &acpAgent{
		sessions: make(map[acp.SessionId]*acpSession),
	}

	cleanupCount := 0
	sessionID := acp.SessionId("test-session")
	_, cancel := context.WithCancel(context.Background())
	agent.sessions[sessionID] = &acpSession{
		sessionCancel: func() {
			cleanupCount++
			cancel()
		},
	}

	// Simulate concurrent Cancel() removing the session first
	agent.Cancel(context.Background(), acp.CancelNotification{SessionId: sessionID})
	assert.Equal(t, 1, cleanupCount, "Cancel should have cleaned up once")

	// cleanupAllSessions should find no sessions left
	agent.cleanupAllSessions()
	assert.Equal(t, 1, cleanupCount, "cleanupAllSessions should not double-cleanup")
}
