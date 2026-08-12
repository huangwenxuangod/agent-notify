// Package tray keeps desktop control actions available after the Wails window
// is hidden. Platform files provide the native menu implementation.
package tray

type Actions struct {
	Open   func()
	Pause  func()
	Resume func()
	Quit   func()
}
