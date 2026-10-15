//
//  virtualization_14.m
//
//  Created by codehex.
//

#import "virtualization_14.h"

/*!
 @abstract Initialize a VZNVMExpressControllerDeviceConfiguration with a device attachment.
 @param attachment The storage device attachment. This defines how the virtualized device operates on the host side.
 @see VZDiskImageStorageDeviceAttachment
 @see https://nvmexpress.org/wp-content/uploads/NVM-Express-1_1b-1.pdf
 */
void *newVZNVMExpressControllerDeviceConfiguration(void *attachment)
{
#ifdef INCLUDE_TARGET_OSX_14
    if (@available(macOS 14, *)) {
        return [[VZNVMExpressControllerDeviceConfiguration alloc] initWithAttachment:(VZStorageDeviceAttachment *)attachment];
    }
#endif
    RAISE_UNSUPPORTED_MACOS_EXCEPTION();
}

/*!
 @abstract Initialize the disk attachment from a file handle.
 @param fileHandle File handle to a block device.
 @param readOnly If YES, the disk attachment is read only, otherwise, if the file handle allows writes, the device can write data into it.
 @param synchronizationMode Defines how the disk synchronizes with the underlying storage when the guest operating system flushes data.
 @param error If not nil, assigned with the error if the initialization failed.
 @return An initialized `VZDiskBlockDeviceStorageDeviceAttachment` or nil if there was an error.
 @discussion
    The file handle is retained by the disk attachment.
    The handle must be open when the virtual machine starts.

    The `readOnly` parameter affects how the disk is exposed to the guest operating system
    by the storage controller. If the disk is intended to be used read-only, it is also recommended
    to open the file handle as read-only.
 */
void *newVZDiskBlockDeviceStorageDeviceAttachment(int fileDescriptor, bool readOnly, int syncMode, void **error)
{
#ifdef INCLUDE_TARGET_OSX_14
    if (@available(macOS 14, *)) {
        NSFileHandle *fileHandle = [[NSFileHandle alloc] initWithFileDescriptor:fileDescriptor];
        return [[VZDiskBlockDeviceStorageDeviceAttachment alloc]
             initWithFileHandle:fileHandle
                       readOnly:(BOOL)readOnly
            synchronizationMode:(VZDiskSynchronizationMode)syncMode
                          error:(NSError *_Nullable *_Nullable)error];
    }
#endif
    RAISE_UNSUPPORTED_MACOS_EXCEPTION();
}