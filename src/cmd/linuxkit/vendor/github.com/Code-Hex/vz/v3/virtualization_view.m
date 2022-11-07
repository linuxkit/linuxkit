//
//  virtualization_view.m
//
//  Created by codehex.
//

#import "virtualization_view.h"

@implementation VZApplication

- (void)run
{
    @autoreleasepool {
        [self finishLaunching];

        shouldKeepRunning = YES;
        do {
            NSEvent *event = [self
                nextEventMatchingMask:NSEventMaskAny
                            untilDate:[NSDate distantFuture]
                               inMode:NSDefaultRunLoopMode
                              dequeue:YES];
            // NSLog(@"event: %@", event);
            [self sendEvent:event];
            [self updateWindows];
        } while (shouldKeepRunning);
    }
}

- (void)terminate:(id)sender
{
    shouldKeepRunning = NO;

    // We should call this method if we want to use `applicationWillTerminate` method.
    //
    // [[NSNotificationCenter defaultCenter]
    //     postNotificationName:NSApplicationWillTerminateNotification
    //                   object:NSApp];

    // This method is used to end up the event loop.
    // If no events are coming, the event loop will always be in a waiting state.
    [self postEvent:self.currentEvent atStart:NO];
}
@end

@implementation AboutViewController

- (instancetype)init
{
    self = [super initWithNibName:nil bundle:nil];
    return self;
}

- (void)loadView
{
    self.view = [NSView new];
    NSImageView *imageView = [NSImageView imageViewWithImage:[NSApp applicationIconImage]];
    NSTextField *appLabel = [self makeLabel:[[NSProcessInfo processInfo] processName]];
    [appLabel setFont:[NSFont boldSystemFontOfSize:16]];
    NSTextField *subLabel = [self makePoweredByLabel];

    NSStackView *stackView = [NSStackView stackViewWithViews:@[
        imageView,
        appLabel,
        subLabel,
    ]];
    [stackView setOrientation:NSUserInterfaceLayoutOrientationVertical];
    [stackView setDistribution:NSStackViewDistributionFillProportionally];
    [stackView setSpacing:10];
    [stackView setAlignment:NSLayoutAttributeCenterX];
    [stackView setContentCompressionResistancePriority:NSLayoutPriorityRequired forOrientation:NSLayoutConstraintOrientationHorizontal];
    [stackView setContentCompressionResistancePriority:NSLayoutPriorityRequired forOrientation:NSLayoutConstraintOrientationVertical];

    [self.view addSubview:stackView];

    [NSLayoutConstraint activateConstraints:@[
        [imageView.widthAnchor constraintEqualToConstant:80], // image size
        [imageView.heightAnchor constraintEqualToConstant:80], // image size
        [stackView.topAnchor constraintEqualToAnchor:self.view.topAnchor
                                            constant:4],
        [stackView.bottomAnchor constraintEqualToAnchor:self.view.bottomAnchor
                                               constant:-16],
        [stackView.leadingAnchor constraintEqualToAnchor:self.view.leadingAnchor
                                                constant:32],
        [stackView.trailingAnchor constraintEqualToAnchor:self.view.trailingAnchor
                                                 constant:-32],
        [stackView.widthAnchor constraintEqualToConstant:300]
    ]];
}

- (NSTextField *)makePoweredByLabel
{
    NSMutableAttributedString *poweredByAttr = [[[NSMutableAttributedString alloc]
        initWithString:@"Powered by "
            attributes:@{
                NSForegroundColorAttributeName : [NSColor labelColor]
            }] autorelease];
    NSURL *repositoryURL = [NSURL URLWithString:@"https://github.com/Code-Hex/vz"];
    NSMutableAttributedString *repository = [self makeHyperLink:@"github.com/Code-Hex/vz" withURL:repositoryURL];
    [poweredByAttr appendAttributedString:repository];
    [poweredByAttr addAttribute:NSFontAttributeName
                          value:[NSFont systemFontOfSize:12]
                          range:NSMakeRange(0, [poweredByAttr length])];

    NSTextField *label = [self makeLabel:@""];
    [label setSelectable:YES];
    [label setAllowsEditingTextAttributes:YES];
    [label setAttributedStringValue:poweredByAttr];
    return label;
}

- (NSTextField *)makeLabel:(NSString *)label
{
    NSTextField *appLabel = [NSTextField labelWithString:label];
    [appLabel setTextColor:[NSColor labelColor]];
    [appLabel setEditable:NO];
    [appLabel setSelectable:NO];
    [appLabel setBezeled:NO];
    [appLabel setBordered:NO];
    [appLabel setBackgroundColor:[NSColor clearColor]];
    [appLabel setAlignment:NSTextAlignmentCenter];
    [appLabel setLineBreakMode:NSLineBreakByWordWrapping];
    [appLabel setUsesSingleLineMode:NO];
    [appLabel setMaximumNumberOfLines:20];
    return appLabel;
}

// https://developer.apple.com/library/archive/qa/qa1487/_index.html
- (NSMutableAttributedString *)makeHyperLink:(NSString *)inString withURL:(NSURL *)aURL
{
    NSMutableAttributedString *attrString = [[NSMutableAttributedString alloc] initWithString:inString];
    NSRange range = NSMakeRange(0, [attrString length]);

    [attrString beginEditing];
    [attrString addAttribute:NSLinkAttributeName value:[aURL absoluteString] range:range];

    // make the text appear in blue
    [attrString addAttribute:NSForegroundColorAttributeName value:[NSColor blueColor] range:range];

    // next make the text appear with an underline
    [attrString addAttribute:NSUnderlineStyleAttributeName
                       value:[NSNumber numberWithInt:NSUnderlineStyleSingle]
                       range:range];

    [attrString endEditing];
    return [attrString autorelease];
}

@end

@implementation AboutPanel

- (instancetype)init
{
    self = [super initWithContentRect:NSZeroRect styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable backing:NSBackingStoreBuffered defer:NO];

    AboutViewController *viewController = [[[AboutViewController alloc] init] autorelease];
    [self setContentViewController:viewController];
    [self setTitleVisibility:NSWindowTitleHidden];
    [self setTitlebarAppearsTransparent:YES];
    [self setBecomesKeyOnlyIfNeeded:NO];
    [self center];
    return self;
}

@end

@implementation AppDelegate {
    VZVirtualMachine *_virtualMachine;
    VZVirtualMachineView *_virtualMachineView;
    CGFloat _windowWidth;
    CGFloat _windowHeight;
}

- (instancetype)initWithVirtualMachine:(VZVirtualMachine *)virtualMachine
                           windowWidth:(CGFloat)windowWidth
                          windowHeight:(CGFloat)windowHeight
{
    self = [super init];
    _virtualMachine = virtualMachine;
    _virtualMachine.delegate = self;

    // Setup virtual machine view configs
    VZVirtualMachineView *view = [[[VZVirtualMachineView alloc] init] autorelease];
    view.capturesSystemKeys = YES;
    view.virtualMachine = _virtualMachine;
    _virtualMachineView = view;

    // Setup some window configs
    _windowWidth = windowWidth;
    _windowHeight = windowHeight;
    return self;
}

/* IMPORTANT: delegate methods are called from VM's queue */
- (void)guestDidStopVirtualMachine:(VZVirtualMachine *)virtualMachine
{
    [NSApp performSelectorOnMainThread:@selector(terminate:) withObject:self waitUntilDone:NO];
}

- (void)virtualMachine:(VZVirtualMachine *)virtualMachine didStopWithError:(NSError *)error
{
    NSLog(@"VM %@ didStopWithError: %@", virtualMachine, error);
    [NSApp performSelectorOnMainThread:@selector(terminate:) withObject:self waitUntilDone:NO];
}

- (void)applicationDidFinishLaunching:(NSNotification *)notification
{
    [self setupMenuBar];
    [self setupGraphicWindow];

    // These methods are required to call here. Because the menubar will be not active even if
    // application is running.
    // See: https://stackoverflow.com/questions/62739862/why-doesnt-activateignoringotherapps-enable-the-menu-bar
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    [NSApp activateIgnoringOtherApps:YES];
}

- (void)windowWillClose:(NSNotification *)notification
{
    [NSApp performSelectorOnMainThread:@selector(terminate:) withObject:self waitUntilDone:NO];
}

- (void)setupGraphicWindow
{
    NSRect rect = NSMakeRect(0, 0, _windowWidth, _windowHeight);
    NSWindow *window = [[[NSWindow alloc] initWithContentRect:rect
                                                    styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable //|NSTexturedBackgroundWindowMask
                                                      backing:NSBackingStoreBuffered
                                                        defer:NO] autorelease];

    [window setOpaque:NO];
    [window setContentView:_virtualMachineView];
    [window setTitleVisibility:NSWindowTitleHidden];
    [window center];

    [window setDelegate:self];
    [window makeKeyAndOrderFront:nil];

    // This code to prevent crash when called applicationShouldTerminateAfterLastWindowClosed.
    // https://stackoverflow.com/a/13470694
    [window setReleasedWhenClosed:NO];
}

- (void)setupMenuBar
{
    NSMenu *menuBar = [[[NSMenu alloc] init] autorelease];
    NSMenuItem *menuBarItem = [[[NSMenuItem alloc] init] autorelease];
    [menuBar addItem:menuBarItem];
    [NSApp setMainMenu:menuBar];

    // App menu
    NSMenu *appMenu = [self setupApplicationMenu];
    [menuBarItem setSubmenu:appMenu];

    // Window menu
    NSMenu *windowMenu = [self setupWindowMenu];
    NSMenuItem *windowMenuItem = [[[NSMenuItem alloc] initWithTitle:@"Window" action:nil keyEquivalent:@""] autorelease];
    [menuBar addItem:windowMenuItem];
    [windowMenuItem setSubmenu:windowMenu];

    // Help menu
    NSMenu *helpMenu = [self setupHelpMenu];
    NSMenuItem *helpMenuItem = [[[NSMenuItem alloc] initWithTitle:@"Help" action:nil keyEquivalent:@""] autorelease];
    [menuBar addItem:helpMenuItem];
    [helpMenuItem setSubmenu:helpMenu];
}

- (NSMenu *)setupApplicationMenu
{
    NSMenu *appMenu = [[[NSMenu alloc] init] autorelease];
    NSString *applicationName = [[NSProcessInfo processInfo] processName];

    NSMenuItem *aboutMenuItem = [[[NSMenuItem alloc]
        initWithTitle:[NSString stringWithFormat:@"About %@", applicationName]
               action:@selector(openAboutWindow:)
        keyEquivalent:@""] autorelease];

    // CapturesSystemKeys toggle
    NSMenuItem *capturesSystemKeysItem = [[[NSMenuItem alloc]
        initWithTitle:@"Enable to send system hot keys to virtual machine"
               action:@selector(toggleCapturesSystemKeys:)
        keyEquivalent:@""] autorelease];
    [capturesSystemKeysItem setState:[self capturesSystemKeysState]];

    // Service menu
    NSMenuItem *servicesMenuItem = [[[NSMenuItem alloc] initWithTitle:@"Services" action:nil keyEquivalent:@""] autorelease];
    NSMenu *servicesMenu = [[[NSMenu alloc] initWithTitle:@"Services"] autorelease];
    [servicesMenuItem setSubmenu:servicesMenu];
    [NSApp setServicesMenu:servicesMenu];

    NSMenuItem *hideOthersItem = [[[NSMenuItem alloc]
        initWithTitle:@"Hide Others"
               action:@selector(hideOtherApplications:)
        keyEquivalent:@"h"] autorelease];
    [hideOthersItem setKeyEquivalentModifierMask:(NSEventModifierFlagOption | NSEventModifierFlagCommand)];

    NSArray *menuItems = @[
        aboutMenuItem,
        [NSMenuItem separatorItem],
        capturesSystemKeysItem,
        [NSMenuItem separatorItem],
        servicesMenuItem,
        [NSMenuItem separatorItem],
        [[[NSMenuItem alloc]
            initWithTitle:[@"Hide " stringByAppendingString:applicationName]
                   action:@selector(hide:)
            keyEquivalent:@"h"] autorelease],
        hideOthersItem,
        [NSMenuItem separatorItem],
        [[[NSMenuItem alloc]
            initWithTitle:[@"Quit " stringByAppendingString:applicationName]
                   action:@selector(terminate:)
            keyEquivalent:@"q"] autorelease],
    ];
    for (NSMenuItem *menuItem in menuItems) {
        [appMenu addItem:menuItem];
    }
    return appMenu;
}

- (NSMenu *)setupWindowMenu
{
    NSMenu *windowMenu = [[[NSMenu alloc] initWithTitle:@"Window"] autorelease];
    NSArray *menuItems = @[
        [[[NSMenuItem alloc] initWithTitle:@"Minimize" action:@selector(performMiniaturize:) keyEquivalent:@"m"] autorelease],
        [[[NSMenuItem alloc] initWithTitle:@"Zoom" action:@selector(performZoom:) keyEquivalent:@""] autorelease],
        [NSMenuItem separatorItem],
        [[[NSMenuItem alloc] initWithTitle:@"Bring All to Front" action:@selector(arrangeInFront:) keyEquivalent:@""] autorelease],
    ];
    for (NSMenuItem *menuItem in menuItems) {
        [windowMenu addItem:menuItem];
    }
    [NSApp setWindowsMenu:windowMenu];
    return windowMenu;
}

- (NSMenu *)setupHelpMenu
{
    NSMenu *helpMenu = [[[NSMenu alloc] initWithTitle:@"Help"] autorelease];
    NSArray *menuItems = @[
        [[[NSMenuItem alloc] initWithTitle:@"Report issue" action:@selector(reportIssue:) keyEquivalent:@""] autorelease],
    ];
    for (NSMenuItem *menuItem in menuItems) {
        [helpMenu addItem:menuItem];
    }
    [NSApp setHelpMenu:helpMenu];
    return helpMenu;
}

- (void)toggleCapturesSystemKeys:(id)sender
{
    NSMenuItem *item = (NSMenuItem *)sender;
    _virtualMachineView.capturesSystemKeys = !_virtualMachineView.capturesSystemKeys;
    [item setState:[self capturesSystemKeysState]];
}

- (NSControlStateValue)capturesSystemKeysState
{
    return _virtualMachineView.capturesSystemKeys ? NSControlStateValueOn : NSControlStateValueOff;
}

- (void)reportIssue:(id)sender
{
    NSString *url = @"https://github.com/Code-Hex/vz/issues/new";
    [[NSWorkspace sharedWorkspace] openURL:[NSURL URLWithString:url]];
}

- (void)openAboutWindow:(id)sender
{
    AboutPanel *aboutPanel = [[[AboutPanel alloc] init] autorelease];
    [aboutPanel makeKeyAndOrderFront:nil];
}
@end
