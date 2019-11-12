package iouring

// #cgo CFLAGS: -g -Wall
// #cgo LDFLAGS: -luring
// #include <liburing.h>
import "C"
import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
	"fmt"
)

type iouring struct {
	sch  chan sche
	ring C.struct_io_uring
}

type sche struct {
	op    opType
	fd    uintptr
	buf   []byte
	pos   int64
	flags uint
	cch   *chan cche
}

type cche struct {
	len int64
	err error
}

const (
	entriesIOUring           = 1024
	entriesSubmissionChannel = 128
	entriesSubmissionBatch   = 128
)

type opType int8

const (
	opRead opType = iota
	opWrite
	opAppend
	opSync
)

func (r *iouring) Init() {
	// TODO: avoid using liburing
	err := C.io_uring_queue_init(entriesIOUring, &r.ring, 0)
	if err < 0 {
		panic(fmt.Sprintf("io_uring initialization failed: error %s", syscall.Errno(-err)))
	}
	r.sch = make(chan sche, entriesSubmissionChannel)
	go r.sloop()
	go r.cloop()
}

func (r *iouring) sloop() {
	for {
		sv := make([]sche, 0, entriesSubmissionBatch)
		s := <-r.sch
		sv = append(sv, s)
	batchloop:
		for len(sv) < cap(sv) {
			select {
			case s := <-r.sch:
				sv = append(sv, s)
			default:
				break batchloop
			}
		}
		r.putsv(sv)
	}
}

func (r *iouring) putsv(sv []sche) {
	// TODO: avoid using liburing
	// TODO: run putsv in a goroutine
	iovecs := make([]C.struct_iovec, len(sv))
	for si, s := range sv {
		sqe := C.io_uring_get_sqe(&r.ring)
		sqe.fd = C.int(s.fd)
		switch s.op {
		case opRead:
			if len(s.buf) > 0 {
				iovecs[si] = C.struct_iovec{unsafe.Pointer(&s.buf[0]), C.ulong(len(s.buf))}
			}
			// C.io_uring_prep_readv(sqe, s.fd, &iovecs[si], 1, s.pos)
			sqe.opcode = C.IORING_OP_READV
			// sqe.off = s.pos
			*(*C.ulong)(unsafe.Pointer(&sqe.anon0[0])) = C.ulong(s.pos)
			sqe.addr = C.ulonglong(uintptr(unsafe.Pointer(&iovecs[si])))
			sqe.len = 1
		case opWrite:
			if len(s.buf) > 0 {
				iovecs[si] = C.struct_iovec{unsafe.Pointer(&s.buf[0]), C.ulong(len(s.buf))}
			}
			// C.io_uring_prep_writev(sqe, s.fd, &iovecs[si], 1, s.pos)
			sqe.opcode = C.IORING_OP_WRITEV
			// sqe.off = s.pos
			*(*C.ulong)(unsafe.Pointer(&sqe.anon0[0])) = C.ulong(s.pos)
			sqe.addr = C.ulonglong(uintptr(unsafe.Pointer(&iovecs[si])))
			sqe.len = 1
		case opAppend:
			if len(s.buf) > 0 {
				iovecs[si] = C.struct_iovec{unsafe.Pointer(&s.buf[0]), C.ulong(len(s.buf))}
			}
			// C.io_uring_prep_writev(sqe, s.fd, &iovecs[si], 1, 0)
			sqe.opcode = C.IORING_OP_WRITEV
			sqe.addr = C.ulonglong(uintptr(unsafe.Pointer(&iovecs[si])))
			sqe.len = 1
			*(*C.__kernel_rwf_t)(unsafe.Pointer(&sqe.anon1[0])) = C.RWF_APPEND
		case opSync:
			// C.io_uring_prep_fsync(sqe, s.fd, s.flags)
			sqe.opcode = C.IORING_OP_FSYNC
			// sqe.fsync_flags = s.flags
			*(*C.uint)(unsafe.Pointer(&sqe.anon1[0])) = C.uint(s.flags)
		default:
			panic("unknown opType")
		}
		// C.io_uring_sqe_set_data(sqe, (uintptr)(unsafe.Pointer(&s.cch)))
		sqe.user_data = C.ulonglong((uintptr)(unsafe.Pointer(s.cch)))
	}
	C.io_uring_submit(&r.ring)
	runtime.KeepAlive(iovecs)
}

func (r *iouring) cloop() {
	// TODO: batch get CQEs in a single syscall
	// TODO: move result parsing and dispatching outside of the wait loop, or in a goroutine
	// TODO: avoid using liburing
	for {
		var cqe *C.struct_io_uring_cqe
		C.io_uring_wait_cqe(&r.ring, &cqe)
		if cqe == nil {
			continue
		}
		var c cche
		if cqe.res >= 0 {
			c.len = int64(cqe.res)
		} else {
			c.err = syscall.Errno(-cqe.res)
		}
		// user_data := C.io_uring_cqe_get_data(cqe)
		user_data := uintptr(cqe.user_data)
		cch := *(*chan cche)(unsafe.Pointer(user_data))
		C.io_uring_cqe_seen(&r.ring, cqe)
		cch <- c
	}
}

func (r *iouring) submitAndWait(op opType, f *os.File, buf []byte, pos int64, flags uint) (int64, error) {
	cch := make(chan cche, 1) // TODO: use a pool?
	r.sch <- sche{op, f.Fd(), buf, pos, flags, &cch}
	c := <-cch
	return c.len, c.err
}

func (r *iouring) ReadFile(f *os.File, buf []byte, pos int64) (int64, error) {
	return r.submitAndWait(opRead, f, buf, pos, 0)
}

func (r *iouring) WriteFile(f *os.File, buf []byte, pos int64) (int64, error) {
	return r.submitAndWait(opWrite, f, buf, pos, 0)
}

func (r *iouring) AppendFile(f *os.File, buf []byte) (int64, error) {
	return r.submitAndWait(opAppend, f, buf, 0, 0)
}

func (r *iouring) SyncFile(f *os.File, flags uint) error {
	_, err := r.submitAndWait(opSync, f, nil, 0, flags)
	return err
}

// global iouring

var (
	global iouring // TODO: sharding?
	once   sync.Once
)

// ReadFile reads from file f into buf starting from position pos.
// Returns the number of bytes read, or an error.
func ReadFile(f *os.File, buf []byte, pos int64) (int64, error) {
	once.Do(global.Init)
	return global.ReadFile(f, buf, pos)
}

// WriteFile writes the contents of buf in file f starting from offset pos.
// Returns the number of bytes written, or an error.
func WriteFile(f *os.File, buf []byte, pos int64) (int64, error) {
	once.Do(global.Init)
	return global.WriteFile(f, buf, pos)
}

// AppendFile appends the contents of buf to the end of file f.
// Returns the number of bytes written, or an error.
func AppendFile(f *os.File, buf []byte) (int64, error) {
	once.Do(global.Init)
	return global.AppendFile(f, buf)
}

// SyncFile performs an fsync with the specified flags on file f.
func SyncFile(f *os.File, flags uint) error {
	once.Do(global.Init)
	return global.SyncFile(f, flags)
}
