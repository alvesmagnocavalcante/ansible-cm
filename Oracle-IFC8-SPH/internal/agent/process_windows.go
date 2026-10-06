package agent

import (
	"context"
	"errors"
	"strings"
	"unsafe"

	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/windows"
)

// Toolhelp reads process names without opening unrelated protected processes.
// Only IFC8 candidates require access to their command line.
func processRunning(ctx context.Context, configPath string) (bool, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(snapshot)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	err = windows.Process32First(snapshot, &entry)
	var readErr error
	for err == nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), "IfcApplication.exe") {
			proc, candidateErr := process.NewProcessWithContext(ctx, int32(entry.ProcessID))
			if candidateErr == nil {
				var args []string
				args, candidateErr = proc.CmdlineSliceWithContext(ctx)
				for _, arg := range args {
					if strings.EqualFold(arg, configPath) {
						return true, nil
					}
				}
			}
			readErr = errors.Join(readErr, candidateErr)
		}
		err = windows.Process32Next(snapshot, &entry)
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return false, errors.Join(readErr, err)
	}
	return false, readErr
}
