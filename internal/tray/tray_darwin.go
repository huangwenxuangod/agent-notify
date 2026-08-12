//go:build darwin

package tray

/*
#cgo LDFLAGS: -framework Cocoa -framework UniformTypeIdentifiers
void agentNotifyInstallTray(void);
void agentNotifyRemoveTray(void);
*/
import "C"

import "sync"

var (
	actions Actions
	mu      sync.RWMutex
)

func Start(next Actions) {
	mu.Lock()
	actions = next
	mu.Unlock()
	C.agentNotifyInstallTray()
}

func Quit() { C.agentNotifyRemoveTray() }

func invoke(action func(Actions)) {
	mu.RLock()
	current := actions
	mu.RUnlock()
	go action(current)
}

//export agentNotifyTrayOpen
func agentNotifyTrayOpen() { invoke(func(a Actions) { if a.Open != nil { a.Open() } }) }

//export agentNotifyTrayPause
func agentNotifyTrayPause() { invoke(func(a Actions) { if a.Pause != nil { a.Pause() } }) }

//export agentNotifyTrayResume
func agentNotifyTrayResume() { invoke(func(a Actions) { if a.Resume != nil { a.Resume() } }) }

//export agentNotifyTrayQuit
func agentNotifyTrayQuit() { invoke(func(a Actions) { if a.Quit != nil { a.Quit() } }) }
