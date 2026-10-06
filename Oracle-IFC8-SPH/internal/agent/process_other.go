//go:build !windows

package agent

import (
	"context"
	"fmt"
)

func processRunning(ctx context.Context, configPath string) (bool, error) {
	return false, fmt.Errorf("monitor IFC8 requer Windows")
}
