//go:build darwin
// +build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void FocusAppWindow() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp activateIgnoringOtherApps:YES];
        [[NSApp mainWindow] makeKeyAndOrderFront:nil];
    });
}
*/
import "C"

func focusAppWindow() {
	C.FocusAppWindow()
}
