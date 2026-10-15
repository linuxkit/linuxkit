#ifndef LINUXKIT_MICROPHONE_PERMISSION_DARWIN_H
#define LINUXKIT_MICROPHONE_PERMISSION_DARWIN_H

enum {
	LKMicrophoneAuthorizationNotDetermined = 0,
	LKMicrophoneAuthorizationRestricted = 1,
	LKMicrophoneAuthorizationDenied = 2,
	LKMicrophoneAuthorizationAuthorized = 3,
};

enum {
	LKMicrophoneRequestPending = 0,
	LKMicrophoneRequestGranted = 1,
	LKMicrophoneRequestDenied = 2,
};

int LKMicrophoneUsageDescriptionAvailable(void);
int LKMicrophoneAuthorizationStatus(void);
void *LKBeginMicrophoneAuthorizationRequest(void);
int LKMicrophoneAuthorizationRequestResult(void *request);
void LKReleaseMicrophoneAuthorizationRequest(void *request);

#endif
