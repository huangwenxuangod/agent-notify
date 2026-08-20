//go:build windows

package tray

import "github.com/getlantern/systray"

func Start(actions Actions) {
	go systray.Run(func() {
		systray.SetTitle("AN")
		systray.SetTooltip("Agent Notify")
		open := systray.AddMenuItem("打开 Agent Notify", "显示控制台")
		pause := systray.AddMenuItem("暂停远程通知 1 小时", "保留系统通知")
		resume := systray.AddMenuItem("恢复远程通知", "恢复远程推送")
		systray.AddSeparator()
		quit := systray.AddMenuItem("退出 Agent Notify", "停止桌面控制台")
		go func() {
			for {
				select {
				case <-open.ClickedCh:
					if actions.Open != nil {
						actions.Open()
					}
				case <-pause.ClickedCh:
					if actions.Pause != nil {
						actions.Pause()
					}
				case <-resume.ClickedCh:
					if actions.Resume != nil {
						actions.Resume()
					}
				case <-quit.ClickedCh:
					if actions.Quit != nil {
						actions.Quit()
					}
					return
				}
			}
		}()
	}, func() {})
}

func Quit() { systray.Quit() }

func ActivateApp() {}

func SystemShutdownRequested() bool { return false }
