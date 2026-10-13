package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// logTrafficLight 把按钮位置追加到诊断文件，便于验证实际布局。
static void logTrafficLight(NSString *message) {
	NSString *path = @"/tmp/git-power-traffic.log";
	NSString *line = [NSString stringWithFormat:@"%@ %@\n", [NSDate date], message];
	NSFileHandle *handle = [NSFileHandle fileHandleForWritingAtPath:path];
	if (!handle) {
		[line writeToFile:path atomically:YES encoding:NSUTF8StringEncoding error:nil];
		return;
	}
	[handle seekToEndOfFile];
	[handle writeData:[line dataUsingEncoding:NSUTF8StringEncoding]];
	[handle closeFile];
}

// findMainWindow 在应用窗口中查找标题匹配的主窗口。
static NSWindow *findMainWindow(void) {
	for (NSWindow *window in [NSApplication sharedApplication].windows) {
		if ([window.title isEqualToString:@"Git Power"]) {
			return window;
		}
	}
	return nil;
}

// centerTrafficLights 用 Auto Layout 把红绿灯中心钉在按钮容器顶边向下
// headerHeight/2 处（即标题栏中线），窗口缩放、全屏后自动保持居中。
static void centerTrafficLights(double headerHeight) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *window = findMainWindow();
		if (!window) {
			logTrafficLight(@"window not found");
			return;
		}
		NSButton *closeBtn = [window standardWindowButton:NSWindowCloseButton];
		NSButton *miniBtn = [window standardWindowButton:NSWindowMiniaturizeButton];
		NSButton *zoomBtn = [window standardWindowButton:NSWindowZoomButton];
		NSView *container = closeBtn.superview;
		if (!closeBtn || !miniBtn || !zoomBtn || !container) {
			logTrafficLight(@"buttons or container missing");
			return;
		}
		logTrafficLight([NSString stringWithFormat:@"window=%@ container=%@ bounds=%@ contentViewBounds=%@",
			window, container, NSStringFromRect(container.bounds), NSStringFromRect(window.contentView.bounds)]);
		NSMutableArray *constraints = [NSMutableArray array];
		for (NSButton *btn in @[closeBtn, miniBtn, zoomBtn]) {
			[btn setTranslatesAutoresizingMaskIntoConstraints:NO];
			[constraints addObject:[NSLayoutConstraint constraintWithItem:btn
				attribute:NSLayoutAttributeCenterY
				relatedBy:NSLayoutRelationEqual
				toItem:container
				attribute:NSLayoutAttributeTop
				multiplier:1
				constant:headerHeight / 2.0]];
		}
		[NSLayoutConstraint activateConstraints:constraints];
	});
}
*/
import "C"

// titleBarHeight 是前端标题栏（h-11）的高度，单位 point。
const titleBarHeight = 44

// centerTrafficLights 让 macOS 红绿灯垂直居中对齐标题栏。
func centerTrafficLights() {
	C.centerTrafficLights(C.double(titleBarHeight))
}
