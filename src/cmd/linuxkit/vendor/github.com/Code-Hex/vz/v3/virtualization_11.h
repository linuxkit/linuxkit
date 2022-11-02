//
//  virtualization_11.h
//
//  Created by codehex.
//

#pragma once

#import "virtualization_helper.h"
#import <Virtualization/Virtualization.h>

/* exported from cgo */
void connectionHandler(void *connection, void *err, void *cgoHandlerPtr);
void changeStateOnObserver(int state, void *cgoHandler);
bool shouldAcceptNewConnectionHandler(void *cgoHandler, void *connection, void *socketDevice);

@interface Observer : NSObject
- (void)observeValueForKeyPath:(NSString *)keyPath ofObject:(id)object change:(NSDictionary *)change context:(void *)context;
@end

/* VZVirtioSocketListener */
@interface VZVirtioSocketListenerDelegateImpl : NSObject <VZVirtioSocketListenerDelegate>
- (instancetype)initWithHandler:(void *)cgoHandler;
- (BOOL)listener:(VZVirtioSocketListener *)listener shouldAcceptNewConnection:(VZVirtioSocketConnection *)connection fromSocketDevice:(VZVirtioSocketDevice *)socketDevice;
@end

/* BootLoader */
void *newVZLinuxBootLoader(const char *kernelPath);
void setCommandLineVZLinuxBootLoader(void *bootLoaderPtr, const char *commandLine);
void setInitialRamdiskURLVZLinuxBootLoader(void *bootLoaderPtr, const char *ramdiskPath);

/* VirtualMachineConfiguration */
bool validateVZVirtualMachineConfiguration(void *config, void **error);
unsigned long long minimumAllowedMemorySizeVZVirtualMachineConfiguration();
unsigned long long maximumAllowedMemorySizeVZVirtualMachineConfiguration();
unsigned int minimumAllowedCPUCountVZVirtualMachineConfiguration();
unsigned int maximumAllowedCPUCountVZVirtualMachineConfiguration();
void *newVZVirtualMachineConfiguration(void *bootLoader,
    unsigned int CPUCount,
    unsigned long long memorySize);
void setEntropyDevicesVZVirtualMachineConfiguration(void *config,
    void *entropyDevices);
void setMemoryBalloonDevicesVZVirtualMachineConfiguration(void *config,
    void *memoryBalloonDevices);
void setNetworkDevicesVZVirtualMachineConfiguration(void *config,
    void *networkDevices);
void *networkDevicesVZVirtualMachineConfiguration(void *config);
void setSerialPortsVZVirtualMachineConfiguration(void *config,
    void *serialPorts);
void setSocketDevicesVZVirtualMachineConfiguration(void *config,
    void *socketDevices);
void *socketDevicesVZVirtualMachineConfiguration(void *config);
void setStorageDevicesVZVirtualMachineConfiguration(void *config,
    void *storageDevices);

/* Configurations */
void *newVZFileHandleSerialPortAttachment(int readFileDescriptor, int writeFileDescriptor);
void *newVZFileSerialPortAttachment(const char *filePath, bool shouldAppend, void **error);
void *newVZVirtioConsoleDeviceSerialPortConfiguration(void *attachment);
void *newVZBridgedNetworkDeviceAttachment(void *networkInterface);
void *newVZNATNetworkDeviceAttachment(void);
void *newVZFileHandleNetworkDeviceAttachment(int fileDescriptor);
void *newVZVirtioNetworkDeviceConfiguration(void *attachment);
void setNetworkDevicesVZMACAddress(void *config, void *macAddress);
void *newVZVirtioEntropyDeviceConfiguration(void);
void *newVZVirtioBlockDeviceConfiguration(void *attachment);
void *newVZDiskImageStorageDeviceAttachment(const char *diskPath, bool readOnly, void **error);
void *newVZVirtioTraditionalMemoryBalloonDeviceConfiguration();
void *newVZVirtioSocketDeviceConfiguration();
void *newVZMACAddress(const char *macAddress);
void *newRandomLocallyAdministeredVZMACAddress();
const char *getVZMACAddressString(void *macAddress);
void *newVZVirtioSocketListener(void *cgoHandlerPtr);
void *VZVirtualMachine_socketDevices(void *machine);
void VZVirtioSocketDevice_setSocketListenerForPort(void *socketDevice, void *vmQueue, void *listener, uint32_t port);
void VZVirtioSocketDevice_removeSocketListenerForPort(void *socketDevice, void *vmQueue, uint32_t port);
void VZVirtioSocketDevice_connectToPort(void *socketDevice, void *vmQueue, uint32_t port, void *cgoHandlerPtr);

/* VirtualMachine */
void *newVZVirtualMachineWithDispatchQueue(void *config, void *queue, void *statusHandler);
bool requestStopVirtualMachine(void *machine, void *queue, void **error);
void startWithCompletionHandler(void *machine, void *queue, void *completionHandler);
void pauseWithCompletionHandler(void *machine, void *queue, void *completionHandler);
void resumeWithCompletionHandler(void *machine, void *queue, void *completionHandler);
bool vmCanStart(void *machine, void *queue);
bool vmCanPause(void *machine, void *queue);
bool vmCanResume(void *machine, void *queue);
bool vmCanRequestStop(void *machine, void *queue);

void *makeDispatchQueue(const char *label);

/* VZVirtioSocketConnection */
typedef struct VZVirtioSocketConnectionFlat {
    uint32_t destinationPort;
    uint32_t sourcePort;
    int fileDescriptor;
} VZVirtioSocketConnectionFlat;

VZVirtioSocketConnectionFlat convertVZVirtioSocketConnection2Flat(void *connection);
