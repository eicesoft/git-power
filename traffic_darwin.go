package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// centerTrafficLights 将红绿灯在 headerHeight 高的标题栏中垂直居中。
// 用 Auto Layout 约束把按钮中心钉在内容视图顶边向下 headerHeight/2 处，
// 窗口缩放或全屏后仍保持居中。
static void centerTrafficLights(void *windowPtr, double headerHeight) {
	NSWindow *window = (__bridge NSWindow *)windowPtr;
	if (!window) {
		return;
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		NSView *contentView = [window contentView];
		NSButton *closeBtn = [window standardWindowButton:NSWindowCloseButton];
		NSButton *miniBtn = [window standardWindowButton:NSWindowMiniaturizeButton];
		NSButton *zoomBtn = [window standardWindowButton:NSWindowZoomButton];
		if (!contentView || !closeBtn || !miniBtn || !zoomBtn) {
			return;
		}
		CGFloat leading = closeBtn.frame.origin.x;
		for (NSButton *btn in @[closeBtn, miniBtn, zoomBtn]) {
			[btn setTranslatesAutoresizingMaskIntoConstraints:NO];
		}
		[NSLayoutConstraint activateConstraints:@[
			[NSLayoutConstraint constraintWithItem:closeBtn
				attribute:NSLayoutAttributeLeading
				relatedBy:NSLayoutRelationEqual
				toItem:contentView
				attribute:NSLayoutAttributeLeading
				multiplier:1
				constant:leading],
			[NSLayoutConstraint constraintWithItem:closeBtn
				attribute:NSLayoutAttributeCenterY
				relatedBy:NSLayoutRelationEqual
				toItem:contentView
				attribute:NSLayoutAttributeTop
				multiplier:1
				constant:headerHeight / 2.0],
		]];
	});
}
*/
import "C"

import (
	"context"
	"unsafe"
)

// titleBarHeight 是前端标题栏（h-11）的高度，单位 point。
const titleBarHeight = 44

// centerTrafficLights 让 macOS 红绿灯垂直居中对齐标题栏。
func centerTrafficLights(ctx context.Context) {
	window, ok := ctx.Value("windowpointer").(unsafe.Pointer)
	if !ok || window == nil {
		return
	}
	C.centerTrafficLights(window, C.double(titleBarHeight))
}
