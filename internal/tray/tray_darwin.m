#import <Cocoa/Cocoa.h>

extern void agentNotifyTrayOpen(void);
extern void agentNotifyTrayPause(void);
extern void agentNotifyTrayResume(void);
extern void agentNotifyTrayQuit(void);

@interface AgentNotifyTrayController : NSObject
@end

@implementation AgentNotifyTrayController
- (void)open:(id)sender { agentNotifyTrayOpen(); }
- (void)pause:(id)sender { agentNotifyTrayPause(); }
- (void)resume:(id)sender { agentNotifyTrayResume(); }
- (void)quit:(id)sender { agentNotifyTrayQuit(); }
@end

static NSStatusItem *agentNotifyStatusItem;
static AgentNotifyTrayController *agentNotifyTrayController;

void agentNotifyInstallTray(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    if (agentNotifyStatusItem != nil) { return; }
    agentNotifyTrayController = [AgentNotifyTrayController new];
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

void agentNotifyRemoveTray(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    if (agentNotifyStatusItem != nil) {
      [[NSStatusBar systemStatusBar] removeStatusItem:agentNotifyStatusItem];
      agentNotifyStatusItem = nil;
    }
  });
}
