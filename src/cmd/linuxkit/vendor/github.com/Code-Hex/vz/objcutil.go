package vz

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"

const char *getNSErrorLocalizedDescription(void *err)
{
	NSString *ld = (NSString *)[(NSError *)err localizedDescription];
	return [ld UTF8String];
}

const char *getNSErrorDomain(void *err)
{
	const char *ret;
	@autoreleasepool {
		NSString *domain = (NSString *)[(NSError *)err domain];
		ret = [domain UTF8String];
	}
	return ret;
}

const char *getNSErrorUserInfo(void *err)
{
	NSDictionary<NSErrorUserInfoKey, id> *ui = [(NSError *)err userInfo];
	NSString *uis = [NSString stringWithFormat:@"%@", ui];
	return [uis UTF8String];
}

NSInteger getNSErrorCode(void *err)
{
	return (NSInteger)[(NSError *)err code];
}

void *makeNSMutableArray(unsigned long cap)
{
	return [[NSMutableArray alloc] initWithCapacity:(NSUInteger)cap];
}

void addNSMutableArrayVal(void *ary, void *val)
{
	[(NSMutableArray *)ary addObject:(NSObject *)val];
}

void *newNSError()
{
	NSError *err = nil;
	return err;
}

bool hasError(void *err)
{
	return (NSError *)err != nil;
}

void *minimumAlloc()
{
	return [[NSMutableData dataWithLength:1] mutableBytes];
}

void releaseNSObject(void* o)
{
	@autoreleasepool {
		[(NSObject*)o release];
	}
}

static inline void startNSThread()
{
	[[NSThread new] start]; // put the runtime into multi-threaded mode
}

static inline void releaseDispatch(void *queue)
{
	dispatch_release((dispatch_queue_t)queue);
}
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

// startNSThread starts NSThread.
func startNSThread() {
	C.startNSThread()
}

// releaseDispatch releases allocated dispatch_queue_t
func releaseDispatch(p unsafe.Pointer) {
	C.releaseDispatch(p)
}

// CharWithGoString makes *Char which is *C.Char wrapper from Go string.
func charWithGoString(s string) *char {
	return (*char)(unsafe.Pointer(C.CString(s)))
}

// Char is a wrapper of C.char
type char C.char

// CString converts *C.char from *Char
func (c *char) CString() *C.char {
	return (*C.char)(c)
}

// String converts Go string from *Char
func (c *char) String() string {
	return C.GoString((*C.char)(c))
}

// Free frees allocated *C.char in Go code
func (c *char) Free() {
	C.free(unsafe.Pointer(c))
}

// pointer indicates any pointers which are allocated in objective-c world.
type pointer struct {
	ptr unsafe.Pointer
}

// Release releases allocated resources in objective-c world.
func (p *pointer) Release() {
	C.releaseNSObject(p.Ptr())
	runtime.KeepAlive(p)
}

// Ptr returns raw pointer.
func (o *pointer) Ptr() unsafe.Pointer {
	if o == nil {
		return nil
	}
	return o.ptr
}

// NSObject indicates NSObject
type NSObject interface {
	Ptr() unsafe.Pointer
}

// NSError indicates NSError.
type NSError struct {
	Domain               string
	Code                 int
	LocalizedDescription string
	UserInfo             string
	pointer
}

// newNSErrorAsNil makes nil NSError in objective-c world.
func newNSErrorAsNil() *pointer {
	p := &pointer{
		ptr: unsafe.Pointer(C.newNSError()),
	}
	return p
}

// hasNSError checks passed pointer is NSError or not.
func hasNSError(nserrPtr unsafe.Pointer) bool {
	return (bool)(C.hasError(nserrPtr))
}

func (n *NSError) Error() string {
	if n == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"Error Domain=%s Code=%d Description=%q UserInfo=%s",
		n.Domain,
		n.Code,
		n.LocalizedDescription,
		n.UserInfo,
	)
}

// TODO(codehex): improvement (3 times called C functions now)
func newNSError(p unsafe.Pointer) *NSError {
	if !hasNSError(p) {
		return nil
	}
	domain := (*char)(C.getNSErrorDomain(p))
	description := (*char)(C.getNSErrorLocalizedDescription(p))
	userInfo := (*char)(C.getNSErrorUserInfo(p))
	return &NSError{
		Domain:               domain.String(),
		Code:                 int(C.getNSErrorCode(p)),
		LocalizedDescription: description.String(),
		UserInfo:             userInfo.String(), // NOTE(codehex): maybe we can convert to map[string]interface{}
	}
}

// convertToNSMutableArray converts to NSMutableArray from NSObject slice in Go world.
func convertToNSMutableArray(s []NSObject) *pointer {
	ln := len(s)
	ary := C.makeNSMutableArray(C.ulong(ln))
	for _, v := range s {
		C.addNSMutableArrayVal(ary, v.Ptr())
	}
	p := &pointer{ptr: ary}
	runtime.SetFinalizer(p, func(self *pointer) {
		self.Release()
	})
	return p
}
