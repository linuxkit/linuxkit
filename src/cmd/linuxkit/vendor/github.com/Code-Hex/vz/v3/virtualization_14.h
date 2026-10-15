//
//  virtualization_14.h
//
//  Created by codehex.
//

#pragma once

// FIXME(codehex): this is dirty hack to avoid clang-format error like below
// "Configuration file(s) do(es) not support C++: /github.com/Code-Hex/vz/.clang-format"
#define NSURLComponents NSURLComponents

#import "virtualization_helper.h"
#import <Virtualization/Virtualization.h>

/* macOS 14 API */
void *newVZNVMExpressControllerDeviceConfiguration(void *attachment);
void *newVZDiskBlockDeviceStorageDeviceAttachment(int fileDescriptor, bool readOnly, int syncMode, void **error);