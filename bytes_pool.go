package bytespool

import (
	"math"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

const (
	minCapacity    = 2
	defaultMinSize = 2
	defaultMaxSize = 8192
)

var defaultCapacityPools = NewCapacityPools(defaultMinSize, defaultMaxSize)

type CapacityPools struct {
	minSize  int
	maxSize  int
	maxIndex int
	pools    []*bytesPool
}

type bytesPool struct {
	capacity int
	pool     sync.Pool
}

// InitDefaultPools initialize to the default pool.
func InitDefaultPools(minSize, maxSize int) {
	defaultCapacityPools = NewCapacityPools(minSize, maxSize)
}

// NewCapacityPools divide into multiple pools according to the capacity scale.
func NewCapacityPools(minSize, maxSize int) *CapacityPools {
	var pools []*bytesPool
	if minSize < minCapacity {
		minSize = minCapacity
	}
	if maxSize < minSize {
		maxSize = minSize
	}

	for i := minSize; i < maxSize; i *= 2 {
		pools = append(pools, newBytesPool(i))
	}
	pools = append(pools, newBytesPool(maxSize))

	return &CapacityPools{
		minSize:  minSize,
		maxSize:  maxSize,
		maxIndex: len(pools) - 1,
		pools:    pools,
	}
}

func newBytesPool(size int) *bytesPool {
	return &bytesPool{
		capacity: size,
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, size, size)
				return &buf
			},
		},
	}
}

// Make return an empty bytes pointer variable.
// Length is 0, default capacity is maxSize.
func (p *CapacityPools) Make(capacity int) []byte {
	return p.New(capacity)[:0]
}

func (p *CapacityPools) Make64(capacity uint64) []byte {
	return p.New(int(capacity))[:0]
}

func (p *CapacityPools) MakeMax() []byte {
	return p.New(p.maxSize)[:0]
}

func (p *CapacityPools) MakeMin() []byte {
	return p.New(p.minSize)[:0]
}

// New return bytes of the specified size.
// Length is size, may contain old data.
func (p *CapacityPools) New(size int) (buf []byte) {
	bp := p.getPool(size)
	if bp == nil {
		return make([]byte, size)
	}

	ptr, _ := bp.pool.Get().(unsafe.Pointer)
	if ptr == nil {
		return make([]byte, size, bp.capacity)
	}

	slice := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	slice.Data = uintptr(ptr)
	slice.Len = size
	slice.Cap = bp.capacity
	runtime.KeepAlive(ptr)
	return
}

func (p *CapacityPools) New64(size uint64) []byte {
	return p.New(int(size))
}

func (p *CapacityPools) NewMax() []byte {
	return p.New(p.maxSize)
}

func (p *CapacityPools) NewMin() []byte {
	return p.New(p.minSize)
}

// Release put it back into the pool of the corresponding scale.
// Discard buffer larger than the maximum capacity.
func (p *CapacityPools) Release(buf []byte) bool {
	n := cap(buf)
	if n == 0 || n > p.maxSize {
		return false
	}
	bp := p.getPool(cap(buf))
	if bp == nil || n != bp.capacity {
		return false
	}
	// array pointer
	bp.pool.Put(unsafe.Pointer(&buf[:1][0]))
	return true
}

func (p *CapacityPools) getPool(size int) *bytesPool {
	if size <= p.minSize {
		return p.pools[0]
	}
	if size == p.maxSize {
		return p.pools[p.maxIndex]
	}
	if size > p.maxSize {
		return nil
	}

	idx := int(math.Ceil(math.Log2(float64(size) / float64(p.minSize))))
	if idx < 0 {
		idx = 0
	}
	if idx > p.maxIndex {
		return nil
	}

	return p.pools[idx]
}

func Make(size int) []byte {
	return defaultCapacityPools.Make(size)
}

func Make64(size uint64) []byte {
	return defaultCapacityPools.Make64(size)
}

func MakeMax() []byte {
	return defaultCapacityPools.MakeMax()
}

func MakeMin() []byte {
	return defaultCapacityPools.MakeMin()
}

func New(size int) []byte {
	return defaultCapacityPools.New(size)
}

func New64(size uint64) []byte {
	return defaultCapacityPools.New64(size)
}

func NewMax() []byte {
	return defaultCapacityPools.NewMax()
}

func NewMin() []byte {
	return defaultCapacityPools.NewMin()
}

func Release(buf []byte) bool {
	return defaultCapacityPools.Release(buf)
}
