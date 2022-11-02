//
//  virtualization_view.h
//
//  Created by codehex.
//

#pragma once

#import <Availability.h>
#import <Cocoa/Cocoa.h>
#import <Virtualization/Virtualization.h>

@interface VZApplication : NSApplication {
    bool shouldKeepRunning;
}
@end

@interface AboutViewController : NSViewController
- (instancetype)init;
@end

@interface AboutPanel : NSPanel
- (instancetype)init;
@end

API_AVAILABLE(macos(12.0))
@interface AppDelegate : NSObject <NSApplicationDelegate, NSWindowDelegate, VZVirtualMachineDelegate>
- (instancetype)initWithVirtualMachine:(VZVirtualMachine *)virtualMachine
                           windowWidth:(CGFloat)windowWidth
                          windowHeight:(CGFloat)windowHeight;
@end