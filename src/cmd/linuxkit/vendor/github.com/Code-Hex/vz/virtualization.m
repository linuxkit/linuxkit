//
//  virtualization.m
//
//  Created by codehex.
//

#import "virtualization.h"

char *copyCString(NSString *nss)
{
    const char *cc = [nss UTF8String];
    char *c = calloc([nss length]+1, 1);
    strncpy(c, cc, [nss length]);
    return c;
}

@implementation Observer
- (void)observeValueForKeyPath:(NSString *)keyPath ofObject:(id)object change:(NSDictionary *)change context:(void *)context;
{
    
    @autoreleasepool {
        if ([keyPath isEqualToString:@"state"]) {
            int newState = (int)[change[NSKeyValueChangeNewKey] integerValue];
            char *vmid = copyCString((NSString *)context);
            changeStateOnObserver(newState, vmid);
            free(vmid);
        } else {
            // bool canVal = (bool)[change[NSKeyValueChangeNewKey] boolValue];
            // char *vmid = copyCString((NSString *)context);
            // char *key = copyCString(keyPath);
            // changeCanPropertyOnObserver(canVal, vmid, key);
            // free(vmid);
            // free(key);
        }
    }
}
@end

/*!
 @abstract Create a VZLinuxBootLoader with the Linux kernel passed as URL.
 @param kernelPath  Path of Linux kernel on the local file system.
*/
void *newVZLinuxBootLoader(const char *kernelPath)
{
    VZLinuxBootLoader *ret;
    @autoreleasepool {
        NSString *kernelPathNSString = [NSString stringWithUTF8String:kernelPath];
        NSURL *kernelURL = [NSURL fileURLWithPath:kernelPathNSString];
        ret = [[VZLinuxBootLoader alloc] initWithKernelURL:kernelURL];
    }
    return ret;
}

/*!
 @abstract Set the command-line parameters.
 @param bootLoader VZLinuxBootLoader
 @param commandLine The command-line parameters passed to the kernel on boot.
 @link https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html
 */
void setCommandLineVZLinuxBootLoader(void *bootLoaderPtr, const char *commandLine)
{
    VZLinuxBootLoader *bootLoader = (VZLinuxBootLoader *)bootLoaderPtr;
    @autoreleasepool {
        NSString *commandLineNSString = [NSString stringWithUTF8String:commandLine];
        [bootLoader setCommandLine:commandLineNSString];
    }
}

/*!
 @abstract Set the optional initial RAM disk.
 @param bootLoader VZLinuxBootLoader
 @param ramdiskPath The RAM disk is mapped into memory before booting the kernel.
 @link https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html
 */
void setInitialRamdiskURLVZLinuxBootLoader(void *bootLoaderPtr, const char *ramdiskPath)
{
    VZLinuxBootLoader *bootLoader = (VZLinuxBootLoader *)bootLoaderPtr;
    @autoreleasepool {
        NSString *ramdiskPathNSString = [NSString stringWithUTF8String:ramdiskPath];
        NSURL *ramdiskURL = [NSURL fileURLWithPath:ramdiskPathNSString];
        [bootLoader setInitialRamdiskURL:ramdiskURL];
    }
}


/*!
 @abstract Validate the configuration.
 @param config  Virtual machine configuration.
 @param error If not nil, assigned with the validation error if the validation failed.
 @return true if the configuration is valid.
 */
bool validateVZVirtualMachineConfiguration(void *config, void **error)
{
    return (bool)[(VZVirtualMachineConfiguration *)config
            validateWithError:(NSError * _Nullable * _Nullable)error];
}

/*!
 @abstract Create a new Virtual machine configuration.
 @param bootLoader Boot loader used when the virtual machine starts.
 
 @param CPUCount Number of CPUs.
 @discussion
    The number of CPUs must be a value between VZVirtualMachineConfiguration.minimumAllowedCPUCount
    and VZVirtualMachineConfiguration.maximumAllowedCPUCount.

 @see VZVirtualMachineConfiguration.minimumAllowedCPUCount
 @see VZVirtualMachineConfiguration.maximumAllowedCPUCount
 
 @param memorySize Virtual machine memory size in bytes.
 @discussion
    The memory size must be a multiple of a 1 megabyte (1024 * 1024 bytes) between VZVirtualMachineConfiguration.minimumAllowedMemorySize
    and VZVirtualMachineConfiguration.maximumAllowedMemorySize.

    The memorySize represents the total physical memory seen by a guest OS running in the virtual machine.
    Not all memory is allocated on start, the virtual machine allocates memory on demand.
 @see VZVirtualMachineConfiguration.minimumAllowedMemorySize
 @see VZVirtualMachineConfiguration.maximumAllowedMemorySize
 */
void *newVZVirtualMachineConfiguration(void *bootLoaderPtr,
                                        unsigned int CPUCount,
                                        unsigned long long memorySize)
{
    VZVirtualMachineConfiguration *config = [[VZVirtualMachineConfiguration alloc] init];
    [config setBootLoader:(VZLinuxBootLoader *)bootLoaderPtr];
    [config setCPUCount:(NSUInteger)CPUCount];
    [config setMemorySize:memorySize];
    return config;
}

/*!
 @abstract List of entropy devices. Empty by default.
 @see VZVirtioEntropyDeviceConfiguration
*/
void setEntropyDevicesVZVirtualMachineConfiguration(void *config,
                                                    void *entropyDevices)
{
    [(VZVirtualMachineConfiguration *)config setEntropyDevices:[(NSMutableArray *)entropyDevices copy]];
}


/*!
 @abstract List of memory balloon devices. Empty by default.
 @see VZVirtioTraditionalMemoryBalloonDeviceConfiguration
*/
void setMemoryBalloonDevicesVZVirtualMachineConfiguration(void *config,
                                                    void *memoryBalloonDevices)
{
    [(VZVirtualMachineConfiguration *)config setMemoryBalloonDevices:[(NSMutableArray *)memoryBalloonDevices copy]];
}

/*!
 @abstract List of network adapters. Empty by default.
 @see VZVirtioNetworkDeviceConfiguration
 */
void setNetworkDevicesVZVirtualMachineConfiguration(void *config,
                                                          void *networkDevices)
{
    [(VZVirtualMachineConfiguration *)config setNetworkDevices:[(NSMutableArray *)networkDevices copy]];
}

/*!
 @abstract List of serial ports. Empty by default.
 @see VZVirtioConsoleDeviceSerialPortConfiguration
 */
void setSerialPortsVZVirtualMachineConfiguration(void *config,
                                                    void *serialPorts)
{
    [(VZVirtualMachineConfiguration *)config setSerialPorts:[(NSMutableArray *)serialPorts copy]];
}


/*!
 @abstract List of socket devices. Empty by default.
 @see VZVirtioSocketDeviceConfiguration
 */
void setSocketDevicesVZVirtualMachineConfiguration(void *config,
                                                 void *socketDevices)
{
    [(VZVirtualMachineConfiguration *)config setSocketDevices:[(NSMutableArray *)socketDevices copy]];
}

/*!
 @abstract List of disk devices. Empty by default.
 @see VZVirtioBlockDeviceConfiguration
 */
void setStorageDevicesVZVirtualMachineConfiguration(void *config,
                                                   void *storageDevices)
{
    [(VZVirtualMachineConfiguration *)config setStorageDevices:[(NSMutableArray *)storageDevices copy]];
}

/*!
 @abstract Intialize the VZFileHandleSerialPortAttachment from file descriptors.
 @param readFileDescriptor File descriptor for reading from the file.
 @param writeFileDescriptor File descriptor for writing to the file.
 @discussion
    Each file descriptor must a valid.
*/
void *newVZFileHandleSerialPortAttachment(int readFileDescriptor, int writeFileDescriptor)
{
    VZFileHandleSerialPortAttachment *ret;
    @autoreleasepool {
        NSFileHandle *fileHandleForReading = [[NSFileHandle alloc] initWithFileDescriptor:readFileDescriptor];
        NSFileHandle *fileHandleForWriting = [[NSFileHandle alloc] initWithFileDescriptor:writeFileDescriptor];
        ret = [[VZFileHandleSerialPortAttachment alloc]
                                       initWithFileHandleForReading:fileHandleForReading
                                       fileHandleForWriting:fileHandleForWriting];
    }
    return ret;
}

/*!
 @abstract Initialize the VZFileSerialPortAttachment from a URL of a file.
 @param filePath The path of the file for the attachment on the local file system.
 @param shouldAppend True if the file should be opened in append mode, false otherwise.
        When a file is opened in append mode, writing to that file will append to the end of it.
 @param error If not nil, used to report errors if intialization fails.
 @return A VZFileSerialPortAttachment on success. Nil otherwise and the error parameter is populated if set.
 */
void *newVZFileSerialPortAttachment(const char *filePath, bool shouldAppend, void **error)
{
    VZFileSerialPortAttachment *ret;
    @autoreleasepool {
        NSString *filePathNSString = [NSString stringWithUTF8String:filePath];
        NSURL *fileURL = [NSURL fileURLWithPath:filePathNSString];
        ret = [[VZFileSerialPortAttachment alloc]
                    initWithURL:fileURL append:(BOOL)shouldAppend error:(NSError * _Nullable * _Nullable)error];
    }
    return ret;
}

/*!
 @abstract Create a new Virtio Console Serial Port Device configuration
 @param attachment Base class for a serial port attachment.
 @discussion
    The device creates a console which enables communication between the host and the guest through the Virtio interface.

    The device sets up a single port on the Virtio console device.
 */
void *newVZVirtioConsoleDeviceSerialPortConfiguration(void *attachment)
{
    VZVirtioConsoleDeviceSerialPortConfiguration *config = [[VZVirtioConsoleDeviceSerialPortConfiguration alloc] init];
    [config setAttachment:(VZSerialPortAttachment *)attachment];
    return config;
}

/*!
 @abstract Create a new Network device attachment bridging a host physical interface with a virtual network device.
 @param networkInterface a network interface that bridges a physical interface.
 @discussion
    A bridged network allows the virtual machine to use the same physical interface as the host. Both host and virtual machine
    send and receive packets on the same physical interface but have distinct network layers.

    The bridge network device attachment is used with a VZNetworkDeviceConfiguration to define a virtual network device.

    Using a VZBridgedNetworkDeviceAttachment requires the app to have the "com.apple.vm.networking" entitlement.

 @see VZBridgedNetworkInterface
 @see VZNetworkDeviceConfiguration
 @see VZVirtioNetworkDeviceConfiguration
 */
void *newVZBridgedNetworkDeviceAttachment(void *networkInterface)
{
    return [[VZBridgedNetworkDeviceAttachment alloc] initWithInterface:(VZBridgedNetworkInterface *)networkInterface];
}

/*!
 @abstract Create a new Network device attachment using network address translation (NAT) with outside networks.
 @discussion
    Using the NAT attachment type, the host serves as router and performs network address translation for accesses to outside networks.

 @see VZNetworkDeviceConfiguration
 @see VZVirtioNetworkDeviceConfiguration
 */
void *newVZNATNetworkDeviceAttachment()
{
    return [[VZNATNetworkDeviceAttachment alloc] init];
}

/*!
 @abstract Create a new Network device attachment sending raw network packets over a file handle.
 @discussion
    The file handle attachment transmits the raw packets/frames between the virtual network interface and a file handle.
    The data transmitted through this attachment is at the level of the data link layer.

    The file handle must hold a connected datagram socket.

 @see VZNetworkDeviceConfiguration
 @see VZVirtioNetworkDeviceConfiguration
 */
void *newVZFileHandleNetworkDeviceAttachment(int fileDescriptor)
{
    VZFileHandleNetworkDeviceAttachment *ret;
    @autoreleasepool {
        NSFileHandle *fileHandle = [[NSFileHandle alloc] initWithFileDescriptor:fileDescriptor];
        ret = [[VZFileHandleNetworkDeviceAttachment alloc] initWithFileHandle:fileHandle];
    }
    return ret;
}

/*!
 @abstract Create  a new Configuration of a paravirtualized network device of type Virtio Network Device.
 @discussion
    The communication channel used on the host is defined through the attachment. It is set with the VZNetworkDeviceConfiguration.attachment property.

    The configuration is only valid with valid MACAddress and attachment.

 @see VZVirtualMachineConfiguration.networkDevices
 
 @param attachment  Base class for a network device attachment.
 @discussion
    A network device attachment defines how a virtual network device interfaces with the host system.

    VZNetworkDeviceAttachment should not be instantiated directly. One of its subclasses should be used instead.

    Common attachment types include:
    - VZNATNetworkDeviceAttachment
    - VZFileHandleNetworkDeviceAttachment

 @see VZBridgedNetworkDeviceAttachment
 @see VZFileHandleNetworkDeviceAttachment
 @see VZNATNetworkDeviceAttachment
 */
void *newVZVirtioNetworkDeviceConfiguration(void *attachment)
{
    VZVirtioNetworkDeviceConfiguration *config = [[VZVirtioNetworkDeviceConfiguration alloc] init];
    [config setAttachment:(VZNetworkDeviceAttachment *)attachment];
    return config;
}

/*!
 @abstract Create a new Virtio Entropy Device confiuration
 @discussion The device exposes a source of entropy for the guest's random number generator.
*/
void *newVZVirtioEntropyDeviceConfiguration()
{
    return [[VZVirtioEntropyDeviceConfiguration alloc] init];
}

/*!
 @abstract Initialize a VZVirtioBlockDeviceConfiguration with a device attachment.
 @param attachment The storage device attachment. This defines how the virtualized device operates on the host side.
 @see VZDiskImageStorageDeviceAttachment
 */
void *newVZVirtioBlockDeviceConfiguration(void *attachment)
{
    return [[VZVirtioBlockDeviceConfiguration alloc] initWithAttachment:(VZStorageDeviceAttachment *)attachment];
}

/*!
 @abstract Initialize the attachment from a local file url.
 @param diskPath Local file path to the disk image in RAW format.
 @param readOnly If YES, the device attachment is read-only, otherwise the device can write data to the disk image.
 @param error If not nil, assigned with the error if the initialization failed.
 @return A VZDiskImageStorageDeviceAttachment on success. Nil otherwise and the error parameter is populated if set.
 */
void *newVZDiskImageStorageDeviceAttachment(const char *diskPath, bool readOnly, void **error)
{
    VZDiskImageStorageDeviceAttachment *ret;
    @autoreleasepool {
        NSString *diskPathNSString = [NSString stringWithUTF8String:diskPath];
        NSURL *diskURL = [NSURL fileURLWithPath:diskPathNSString];
        ret = [[VZDiskImageStorageDeviceAttachment alloc]
            initWithURL:diskURL
            readOnly:(BOOL)readOnly
            error:(NSError * _Nullable * _Nullable)error];
    }
    return ret;
}


/*!
 @abstract Create a configuration of the Virtio traditional memory balloon device.
 @discussion
    This configuration creates a Virtio traditional memory balloon device which allows for managing guest memory.
    Only one Virtio traditional memory balloon device can be used per virtual machine.
 @see VZVirtioTraditionalMemoryBalloonDevice
 */
void *newVZVirtioTraditionalMemoryBalloonDeviceConfiguration()
{
    return [[VZVirtioTraditionalMemoryBalloonDeviceConfiguration alloc] init];
}

/*!
 @abstract Create a configuration of the Virtio socket device.
 @discussion
    This configuration creates a Virtio socket device for the guest which communicates with the host through the Virtio interface.

    Only one Virtio socket device can be used per virtual machine.
 @see VZVirtioSocketDevice
 */
void *newVZVirtioSocketDeviceConfiguration()
{
    return [[VZVirtioSocketDeviceConfiguration alloc] init];
}

/*!
 @abstract Initialize the virtual machine.
 @param config The configuration of the virtual machine.
    The configuration must be valid. Validation can be performed at runtime with [VZVirtualMachineConfiguration validateWithError:].
    The configuration is copied by the initializer.
 @param queue The serial queue on which the virtual machine operates.
    Every operation on the virtual machine must be done on that queue. The callbacks and delegate methods are invoked on that queue.
    If the queue is not serial, the behavior is undefined.
 */
void *newVZVirtualMachineWithDispatchQueue(void *config, void *queue, const char *vmid)
{
    VZVirtualMachine *vm = [[VZVirtualMachine alloc]
                initWithConfiguration:(VZVirtualMachineConfiguration *)config
                queue:(dispatch_queue_t)queue];
    @autoreleasepool {
        Observer *o = [[Observer alloc] init];
        NSString *str = [NSString stringWithUTF8String:vmid];
        [vm addObserver:o forKeyPath:@"state"
                options:NSKeyValueObservingOptionNew
                context:[str copy]];
    }
    return vm;
}

/*!
 @abstract Initialize the VZMACAddress from a string representation of a MAC address.
 @param string
    The string should be formatted representing the 6 bytes in hexadecimal separated by a colon character.
        e.g. "01:23:45:ab:cd:ef"

    The alphabetical characters can appear lowercase or uppercase.
 @return A VZMACAddress or nil if the string is not formatted correctly.
 */
void *newVZMACAddress(const char *macAddress)
{
    VZMACAddress *ret;
    @autoreleasepool {
        NSString *str = [NSString stringWithUTF8String:macAddress];
        ret = [[VZMACAddress alloc] initWithString:str];
    }
    return ret;
}

/*!
 @abstract Create a valid, random, unicast, locally administered address.
 @discussion The generated address is not guaranteed to be unique.
 */
void *newRandomLocallyAdministeredVZMACAddress()
{
    return [VZMACAddress randomLocallyAdministeredAddress];
}

/*!
 @abstract Sets the media access control address of the device.
 */
void setNetworkDevicesVZMACAddress(void *config, void *macAddress)
{
    [(VZNetworkDeviceConfiguration *)config setMACAddress:[(VZMACAddress *)macAddress copy]];
}

/*!
 @abstract The address represented as a string.
 @discussion
    The 6 bytes are represented in hexadecimal form, separated by a colon character.
    Alphabetical characters are lowercase.

    The address is compatible with the parameter of -[VZMACAddress initWithString:].
 */
const char *getVZMACAddressString(void *macAddress)
{
    return [[(VZMACAddress *)macAddress string] UTF8String];
}

/*!
 @abstract Request that the guest turns itself off.
 @param error If not nil, assigned with the error if the request failed.
 @return YES if the request was made successfully.
 */
bool requestStopVirtualMachine(void *machine, void *queue, void **error)
{
    __block BOOL ret;
    dispatch_sync((dispatch_queue_t)queue, ^{
        ret = [(VZVirtualMachine *)machine requestStopWithError:(NSError * _Nullable *_Nullable)error];
    });
    return (bool)ret;
}

void *makeDispatchQueue(const char *label)
{
    //dispatch_queue_attr_t attr = dispatch_queue_attr_make_with_qos_class(DISPATCH_QUEUE_SERIAL, QOS_CLASS_DEFAULT, 0);
    dispatch_queue_t queue = dispatch_queue_create(label, DISPATCH_QUEUE_SERIAL);
    //dispatch_retain(queue);
    return queue;
}

typedef void (^handler_t)(NSError *);

handler_t generateHandler(const char *vmid, void handler(void *, char *))
{
    handler_t ret;
    @autoreleasepool {
        NSString *str = [NSString stringWithUTF8String:vmid];
        ret = Block_copy(^(NSError *err){
            handler(err, copyCString(str));
        });
    }
    return ret;
}

void startWithCompletionHandler(void *machine, void *queue, const char *vmid)
{
    handler_t handler = generateHandler(vmid, startHandler);
    dispatch_sync((dispatch_queue_t)queue, ^{
        [(VZVirtualMachine *)machine startWithCompletionHandler:handler];
    });
    Block_release(handler);
}

void pauseWithCompletionHandler(void *machine, void *queue, const char *vmid)
{
    handler_t handler = generateHandler(vmid, pauseHandler);
    dispatch_sync((dispatch_queue_t)queue, ^{
        [(VZVirtualMachine *)machine pauseWithCompletionHandler:handler];
    });
    Block_release(handler);
}

void resumeWithCompletionHandler(void *machine, void *queue, const char *vmid)
{
    handler_t handler = generateHandler(vmid, pauseHandler);
    dispatch_sync((dispatch_queue_t)queue, ^{
        [(VZVirtualMachine *)machine resumeWithCompletionHandler:handler];
    });
    Block_release(handler);
}

// TODO(codehex): use KVO
bool vmCanStart(void *machine, void *queue)
{
    __block BOOL result;
    dispatch_sync((dispatch_queue_t)queue, ^{
        result = ((VZVirtualMachine *)machine).canStart;
    });
    return (bool)result;
}

bool vmCanPause(void *machine, void *queue)
{
    __block BOOL result;
    dispatch_sync((dispatch_queue_t)queue, ^{
        result = ((VZVirtualMachine *)machine).canPause;
    });
    return (bool)result;
}

bool vmCanResume(void *machine, void *queue)
{
    __block BOOL result;
    dispatch_sync((dispatch_queue_t)queue, ^{
        result = ((VZVirtualMachine *)machine).canResume;
    });
    return (bool)result;
}

bool vmCanRequestStop(void *machine, void *queue)
{
    __block BOOL result;
    dispatch_sync((dispatch_queue_t)queue, ^{
        result = ((VZVirtualMachine *)machine).canRequestStop;
    });
    return (bool)result;
}
// --- TODO end
