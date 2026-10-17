//go:build darwin && cgo

package titlebar

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>
#import <objc/runtime.h>

static CGFloat pl_x, pl_center, pl_height;
static char pl_observed;

static void pl_place(NSWindow *w) {
	if ((w.styleMask & NSWindowStyleMaskFullScreen) != 0) {
		return;
	}

	NSButton *close = [w standardWindowButton:NSWindowCloseButton];
	NSButton *mini = [w standardWindowButton:NSWindowMiniaturizeButton];
	NSButton *zoom = [w standardWindowButton:NSWindowZoomButton];

	if (close == nil || mini == nil || zoom == nil) {
		return;
	}

	NSView *container = close.superview.superview;

	if (container == nil) {
		return;
	}

	NSRect r = container.frame;
	r.size.height = pl_height;
	r.origin.y = w.frame.size.height - pl_height;
	container.frame = r;

	CGFloat space = mini.frame.origin.x - close.frame.origin.x;
	NSArray<NSButton *> *buttons = @[close, mini, zoom];

	for (NSUInteger i = 0; i < buttons.count; i++) {
		NSButton *b = buttons[i];
		CGFloat h = b.frame.size.height;
		CGFloat y = b.superview.isFlipped ? pl_center - h / 2 : b.superview.frame.size.height - pl_center - h / 2;
		b.hidden = NO;
		[b setFrameOrigin:NSMakePoint(pl_x + i * space, round(y))];
	}
}

static void pl_attach(CFTypeRef viewRef, CGFloat x, CGFloat center, CGFloat height) {
	NSView *view = (__bridge NSView *)viewRef;

	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *w = view.window;

		if (w == nil) {
			return;
		}

		pl_x = x;
		pl_center = center;
		pl_height = height;

		if (objc_getAssociatedObject(w, &pl_observed) == nil) {
			objc_setAssociatedObject(w, &pl_observed, @YES, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
			NSArray<NSNotificationName> *names = @[
				NSWindowDidResizeNotification,
				NSWindowDidExitFullScreenNotification,
				NSWindowDidBecomeKeyNotification,
				NSWindowDidResignKeyNotification,
			];

			for (NSNotificationName name in names) {
				[NSNotificationCenter.defaultCenter addObserverForName:name object:w queue:nil usingBlock:^(NSNotification *n) {
					pl_place((NSWindow *)n.object);
				}];
			}
		}

		pl_place(w);
	});
}
*/
import "C"

const (
	Native = true
	Width  = 64
)

func Attach(view uintptr, x, center, height float32) {
	if view == 0 {
		return
	}

	C.pl_attach(C.CFTypeRef(view), C.CGFloat(x), C.CGFloat(center), C.CGFloat(height))
}
