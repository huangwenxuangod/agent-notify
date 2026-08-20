#import <Cocoa/Cocoa.h>

extern void agentNotifyTrayOpen(void);
extern void agentNotifyTrayPause(void);
extern void agentNotifyTrayResume(void);
extern void agentNotifyTrayQuit(void);

@interface AgentNotifyTrayController : NSObject
@end

static NSStatusItem *agentNotifyStatusItem;
static AgentNotifyTrayController *agentNotifyTrayController;
static BOOL agentNotifyPowerOffRequested;

@implementation AgentNotifyTrayController
- (void)open:(id)sender { agentNotifyTrayOpen(); }
- (void)pause:(id)sender { agentNotifyTrayPause(); }
- (void)resume:(id)sender { agentNotifyTrayResume(); }
- (void)quit:(id)sender { agentNotifyTrayQuit(); }
- (void)applicationQuit:(id)sender {
  (void)sender;
  agentNotifyTrayQuit();
}
- (void)systemWillPowerOff:(NSNotification *)notification {
  (void)notification;
  agentNotifyPowerOffRequested = YES;
}
@end

void agentNotifyInstallTray(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    if (agentNotifyStatusItem != nil) { return; }
    agentNotifyTrayController = [AgentNotifyTrayController new];
    [[[NSWorkspace sharedWorkspace] notificationCenter]
      addObserver:agentNotifyTrayController
      selector:@selector(systemWillPowerOff:)
      name:NSWorkspaceWillPowerOffNotification
      object:nil];
    // Wails routes the default Quit menu through the same close callback as
    // the window. Forward it through Go so explicit app quits are not hidden.
    NSMenu *mainMenu = [NSApp mainMenu];
    NSMenuItem *appMenuItem = [mainMenu itemAtIndex:0];
    NSMenu *appMenu = [appMenuItem submenu];
    for (NSMenuItem *item in [appMenu itemArray]) {
      if ([[item title] hasPrefix:@"Quit "]) {
        [item setTarget:agentNotifyTrayController];
        [item setAction:@selector(applicationQuit:)];
        break;
      }
    }
    agentNotifyStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    NSButton *button = agentNotifyStatusItem.button;
    if (@available(macOS 11.0, *)) {
      button.image = [NSImage imageWithSystemSymbolName:@"bell.badge" accessibilityDescription:@"Agent Notify"];
    } else {
      button.title = @"AN";
    }
    button.toolTip = @"Agent Notify";
    NSMenu *menu = [NSMenu new];
    [menu addItemWithTitle:@"打开 Agent Notify" action:@selector(open:) keyEquivalent:@""];
    [menu addItemWithTitle:@"暂停远程通知 1 小时" action:@selector(pause:) keyEquivalent:@""];
    [menu addItemWithTitle:@"恢复远程通知" action:@selector(resume:) keyEquivalent:@""];
    [menu addItem:[NSMenuItem separatorItem]];
    [menu addItemWithTitle:@"退出 Agent Notify" action:@selector(quit:) keyEquivalent:@""];
    for (NSMenuItem *item in menu.itemArray) { item.target = agentNotifyTrayController; }
    agentNotifyStatusItem.menu = menu;
  });
}

// agentNotifyActivateApp brings a tray-launched instance to the foreground.
// WindowShow alone is insufficient when the app was started hidden at login.
void agentNotifyActivateApp(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [NSApp activateIgnoringOtherApps:YES];
    NSWindow *window = [NSApp keyWindow];
    if (window == nil && NSApp.windows.count > 0) {
      window = NSApp.windows[0];
    }
    [window makeKeyAndOrderFront:nil];
  });
}

int agentNotifySystemShutdownRequested(void) {
  return agentNotifyPowerOffRequested ? 1 : 0;
}

void agentNotifyRemoveTray(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    if (agentNotifyStatusItem != nil) {
      [[NSStatusBar systemStatusBar] removeStatusItem:agentNotifyStatusItem];
      agentNotifyStatusItem = nil;
    }
  });
}
