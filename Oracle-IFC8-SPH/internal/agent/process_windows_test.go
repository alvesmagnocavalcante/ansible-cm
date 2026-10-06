package agent

import (
	"context"
	"testing"
)

func TestProcessEnumeration(t *testing.T) {
	running, err := processRunning(context.Background(), `C:\unlikely-test-instance\IFC.xml`)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("unexpected IFC8 test instance")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := processRunning(ctx, `C:\IFC.xml`); err == nil {
		t.Fatal("cancellation ignored")
	}
}
