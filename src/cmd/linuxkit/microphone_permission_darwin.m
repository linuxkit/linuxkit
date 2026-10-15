#import <AVFoundation/AVFoundation.h>
#import <Foundation/Foundation.h>

#import "microphone_permission_darwin.h"

@interface LKMicrophoneAuthorizationRequest : NSObject {
@private
	NSLock *_lock;
	NSInteger _result;
}

- (void)completeWithGrant:(BOOL)granted;
- (NSInteger)result;

@end

@implementation LKMicrophoneAuthorizationRequest

- (instancetype)init {
	self = [super init];
	if (self != nil) {
		_lock = [[NSLock alloc] init];
		_result = LKMicrophoneRequestPending;
	}
	return self;
}

- (void)completeWithGrant:(BOOL)granted {
	[_lock lock];
	_result = granted ? LKMicrophoneRequestGranted : LKMicrophoneRequestDenied;
	[_lock unlock];
}

- (NSInteger)result {
	[_lock lock];
	NSInteger result = _result;
	[_lock unlock];
	return result;
}

- (void)dealloc {
	[_lock release];
	[super dealloc];
}

@end

int LKMicrophoneUsageDescriptionAvailable(void) {
	@autoreleasepool {
		id description = [[NSBundle mainBundle] objectForInfoDictionaryKey:@"NSMicrophoneUsageDescription"];
		if (![description isKindOfClass:[NSString class]]) {
			return 0;
		}
		return [(NSString *)description stringByTrimmingCharactersInSet:
			[NSCharacterSet whitespaceAndNewlineCharacterSet]].length > 0;
	}
}

int LKMicrophoneAuthorizationStatus(void) {
	@autoreleasepool {
		return (int)[AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeAudio];
	}
}

void *LKBeginMicrophoneAuthorizationRequest(void) {
	@autoreleasepool {
		LKMicrophoneAuthorizationRequest *request = [[LKMicrophoneAuthorizationRequest alloc] init];
		if (request == nil) {
			return NULL;
		}

		[AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL granted) {
			[request completeWithGrant:granted];
		}];

		return request;
	}
}

int LKMicrophoneAuthorizationRequestResult(void *request) {
	if (request == NULL) {
		return -1;
	}
	return (int)[(LKMicrophoneAuthorizationRequest *)request result];
}

void LKReleaseMicrophoneAuthorizationRequest(void *request) {
	[(LKMicrophoneAuthorizationRequest *)request release];
}
