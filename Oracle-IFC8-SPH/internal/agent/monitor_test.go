package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const healthyLog = `<MonItem ObjType="Ifc" Type="StateLink">Alive</MonItem>
<MonItem Type="StateComm" ObjType="Ifc">Sync</MonItem>
<MonItem ObjType="Pms" Type="StateLink">Alive</MonItem>
<MonItem ObjType="Pms" Type="StateComm">Sync</MonItem>`

func TestLogHealth(t *testing.T) {
	cases := []struct{ name, log, status string }{
		{"healthy", healthyLog, "ONLINE"},
		{"missing", "", "INDETERMINADO"},
		{"offline", healthyLog + `<MonItem ObjType="Pms" Type="StateComm">Off</MonItem>`, "OFFLINE"},
		{"recovery", healthyLog + `<MonItem ObjType="Ifc" Type="StateLink">End</MonItem><MonItem ObjType="Ifc" Type="StateLink">Alive</MonItem>`, "ONLINE"},
		{"partial write", healthyLog + `<MonItem ObjType="Pms" Type="StateComm">Off`, "ONLINE"},
		{"other object", healthyLog + `<MonItem ObjType="Other" Type="StateComm">Off</MonItem>`, "ONLINE"},
		{"xml entity", strings.Replace(healthyLog, ">Alive<", ">Al&#105;ve<", 1), "ONLINE"},
		{"malformed", `<MonItem ObjType="Ifc" Type="StateLink">&invalid;</MonItem>`, "INDETERMINADO"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := healthFromStates(logStates([]byte(tc.log))); got.status != tc.status {
				t.Fatalf("%+v", got)
			}
		})
	}
}

func TestInspectHealth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "IFC.evt")
	cfg := config{logPath: path}
	now := time.Now()
	if got := inspectHealth(cfg, false, nil, now); got.status != "OFFLINE" {
		t.Fatal(got)
	}
	if got := inspectHealth(cfg, false, errors.New("access denied"), now); got.status != "INDETERMINADO" {
		t.Fatal(got)
	}
	if got := inspectHealth(cfg, true, nil, now); got.status != "INDETERMINADO" {
		t.Fatal(got)
	}
	if err := os.WriteFile(path, []byte(healthyLog), 0600); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if got := inspectHealth(cfg, true, nil, now); got.status != "ONLINE" {
		t.Fatal(got)
	}
	cfg.maxLogAge = time.Minute
	if got := inspectHealth(cfg, true, nil, now); got.status != "INDETERMINADO" {
		t.Fatal(got)
	}
	cfg.maxLogAge = 0
	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(16*1024*1024 + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if got := inspectHealth(cfg, true, nil, now); got.status != "INDETERMINADO" {
		t.Fatal(got)
	}
}

func TestAlertDeliveryAndRecovery(t *testing.T) {
	state := alertState{}
	cfg := config{repeat: 5 * time.Minute}
	now := time.Now()
	online, offline := health{"ONLINE", "Alive/Sync"}, health{"OFFLINE", "Off"}
	calls := 0
	fail := false
	send := func(context.Context, string) error {
		calls++
		if fail {
			return errors.New("failed")
		}
		return nil
	}
	notify := func(h health, when time.Time) error {
		return state.notify(context.Background(), h, cfg, "PC", when, send)
	}
	notify(online, now)
	if calls != 0 {
		t.Fatal("initial online notification")
	}
	fail = true
	if notify(offline, now) == nil || state.status != "" {
		t.Fatal("failed delivery confirmed")
	}
	fail = false
	notify(offline, now.Add(30*time.Second))
	notify(offline, now.Add(time.Minute))
	if calls != 2 {
		t.Fatalf("duplicate alert: %d", calls)
	}
	notify(offline, now.Add(6*time.Minute))
	if calls != 3 {
		t.Fatal("missing reminder")
	}
	fail = true
	notify(online, now.Add(7*time.Minute))
	if state.status != "OFFLINE" {
		t.Fatal("failed recovery confirmed")
	}
	fail = false
	notify(online, now.Add(8*time.Minute))
	if state.status != "ONLINE" || calls != 5 {
		t.Fatal("missing recovery")
	}
	cfg.repeat = 0
	notify(offline, now.Add(9*time.Minute))
	notify(offline, now.Add(24*time.Hour))
	if calls != 6 {
		t.Fatal("repeat=0 ignored")
	}
}
