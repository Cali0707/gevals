package llmagent

import (
	"sync"
	"testing"
)

// TestMcpClientRaceCondition reproduces the race between client initialization
// and the ToolListChangedHandler callback firing before session is assigned.
func TestMcpClientRaceCondition(t *testing.T) {
	mc := &mcpClient{}
	ctx := t.Context()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = mc.reloadTools(ctx)
	}()
	wg.Wait()
}

func TestMcpClientConcurrentAccess(t *testing.T) {
	mc := &mcpClient{}
	ctx := t.Context()

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		_ = mc.reloadTools(ctx)
	}()
	go func() {
		defer wg.Done()
		_, _ = mc.CallTool(ctx, "test", nil)
	}()
	go func() {
		defer wg.Done()
		_ = mc.Close()
	}()
	wg.Wait()
}
