/*
Package shm implements shared virtual memory manager.

The Shared Memory is divided into evenly size pages. Each page hase its
own access permissions.

*/
package shm

import (
	"container/list"
	"fmt"
	"sync"
)

type Prot byte

const (
	PROT_READ  = Prot(0x1) /* Page can be read.  */
	PROT_WRITE = Prot(0x2) /* Page can be written.  */
	PROT_EXEC  = Prot(0x4) /* Page can be executed.  */
	PROT_NONE  = Prot(0x0) /* Page can not be accessed.  */
)

////////////////////////////////////////////////////////////////////////////////////////////
// <ERROR DEFINITIONS>

// These type definitions allow the application to explicitly check
// for the kind of error that occurred. Each API call below lists the
// errors that it is allowed to raise.
//
// Also see:
// https://blog.golang.org/error-handling-and-go
// https://blog.golang.org/errors-are-values

// UknownError conatins a message about the error
type UknownError string

func (e UknownError) Error() string {
	return fmt.Sprintf("Shm: Unexpected Error [%s]", string(e))
}

// InvalidAddress conatins the invalid pointer
type InvalidAddress int

func (e InvalidAddress) Error() string {
	return fmt.Sprintf("Shm: Invalid address pointer [%d]", int(e))
}

// InsufficientMemory contains largest free segment remaining.
type InsufficientMemory int

func (e InsufficientMemory) Error() string {
	return fmt.Sprintf("Shm: Largest free block is [%d]", int(e))
}

// OutofBounds contains the address out of bounds
type OutofBounds int

func (e OutofBounds) Error() string {
	return fmt.Sprintf("Shm: Out of bounds address [%d]", int(e))
}

// AccessViolation contains the current operation
type AccessViolation int

func (e AccessViolation) Error() string {
	return fmt.Sprintf("Shm: Could not acesses memory for [%d]", Prot(e))
}

// </ERROR DEFINITIONS>
////////////////////////////////////////////////////////////////////////////////////////////

// SEGVHandler is function signature of the segment violation handler
type SEGVHandler func(addr int, length int, fault Prot) error

// VSM implements the Virtual Shared Memory interface used by the DSM
type VSM interface {
	// Allocates a block of free memory and returns the start of that block
	// Can return the following errors:
	// - InsufficientMemory
	Malloc(size int) (int, error)

	// Frees a previously allocated block of memory.
	// Can return the following errors:
	// - InvalidAddress
	// -
	Free(addr int) error

	// MProtect changes the access protections for the pages that span the memory block
	// Can return the following errors:
	// - InvalidAddress
	// - OutOfBounds
	MProtect(addr int, len int, prot byte) error

	// Get the protection status of a page
	GetProtStatus(page int) Prot

	// Get the protection status of a list of pages
	GetProtStatusPages(pages []int) []Prot

	// Install a SEGV handler to handle page access faults
	// Can return the following errors:
	// -
	InstallSEGVHandler(handler SEGVHandler) error

	// Read a single byte at the specified address or Read array of bytes
	// Can return the following errors:
	// -
	Read(addr int) (byte, error)
	ReadN(addr int, len int) ([]byte, error)

	// Write a byte at address or write an array of bytes starting at address
	// Can return the following errors:
	// -
	Write(addr int, val byte) error
	WriteN(addr int, data []byte) error

	// Read a page. This is used by the DSM library to get a whole page
	ReadPage(page int) []byte

	// Write a page. This is used by the DSM library to write a page
	WritePage(page int, data []byte)

	// GetSize gets the size of the Shared Memory
	GetSize() int

	// Get page size
	GetPageSize() int

	// Get page for an address
	GetPage(addr int) int

	// GetPages gets a list of pages that span a block of memory
	// Can return the following errors:
	// - OutofBounds
	GetPages(start int, length int) ([]int, error)
}

// Chunk is a chunk of memory on the free list
type Chunk struct {
	start int
	end   int
}

// Shm is the structure that holds all of the Shared Virtual Memory attributes
type Shm struct {
	mem        []byte        // block of shared memory
	size       int           // total size of memory
	pageSize   int           // Size of each page
	pageAccess []Prot        // Protection status of the page
	handlers   []SEGVHandler // List of handlers registered
	freeList   *list.List    // List of free chunks
	mallocs    map[int]int   // map of start of allocated block to length
	rwMutex    *sync.Mutex   // mutex to protect RW access

}

// NewShm creates a new Shm object to manage the given size of memory
// Shm object is initialized with one chunk of free memory spanning the entire
// size of the memory
func NewShm(pageSize int, numPages int) *Shm {
	s := new(Shm)
	s.size = numPages * pageSize
	s.mem = make([]byte, s.size)
	s.pageSize = pageSize
	s.pageAccess = make([]Prot, numPages)
	s.handlers = make([]SEGVHandler, 0)
	s.freeList = list.New()
	s.freeList.PushFront(Chunk{0, s.size - 1})
	s.mallocs = make(map[int]int)
	s.rwMutex = new(sync.Mutex)
	return s
}

func (s *Shm) DumpFreeList() {
	i := 0
	for e := s.freeList.Front(); e != nil; e = e.Next() {
		chunk := e.Value.(Chunk)
		fmt.Print(i, ":{", chunk.start, chunk.end, "},")
		i++
		if i > 10 {
			break
		}
	}
	fmt.Println()
}

func (s *Shm) DumpAllocList() {
	for addr, len := range s.mallocs {
		fmt.Print("{", addr, len, "},")
	}
	fmt.Println()
}

func (s *Shm) DumpPages() {
	for i := 0; i < s.size/s.pageSize; i++ {
		fmt.Print("|")
		count := 0
		for bytes := 0; bytes < s.pageSize; bytes++ {
			fmt.Print(s.mem[bytes+i*s.pageSize], "-")
			count++
		}
		fmt.Print("|")
		fmt.Println(" Page Len:", count)
	}
}

func (s *Shm) DumpPage(page int) string {
	pageStart := page * s.pageSize
	var out string
	if pageStart > s.size {
		return out
	}
	for a := pageStart; a < pageStart+s.pageSize; a++ {
		if a%32 == 0 {
			out += fmt.Sprintf("\n%04X: ", a)
		}
		out += fmt.Sprintf(" %02X", s.mem[a])
	}
	return out + "\n"
}

// Malloc a block of free memory and returns the start of that block
// Return OutofBounds error if attempting to malloc 0 or less bytes.
func (s *Shm) Malloc(size int) (int, error) {
	// walk the free list an allocate memory block from first chunk that has
	// enough space. Adjust the chunk size and add the allocated block to
	// the malloc map
	if size < 0 {
		return -1, OutofBounds(size)
	}

	max := 0
	for e := s.freeList.Front(); e != nil; e = e.Next() {
		chunk := e.Value.(Chunk)
		chunkSize := chunk.end - chunk.start + 1

		// Keep track of the size of the maximum free chunk
		if max < chunkSize {
			max = chunkSize
		}

		// Allocate memory if the block fits in this chunk
		if size < chunkSize {
			//Sufficient space, malloc space and adjust free chunk
			s.mallocs[chunk.start] = size
			e.Value = Chunk{start: chunk.start + size, end: chunk.end}
			return chunk.start, nil
		} else if size == chunkSize {
			s.mallocs[chunk.start] = size
			s.freeList.Remove(e)
			return chunk.start, nil
		}
	}
	return -1, InsufficientMemory(max)
}

// Free a previously allocated block of memory.
func (s *Shm) Free(start int) error {
	// First check if this is a valid address and remove it from the
	// mallocs map and increase the size of the free record or insert it
	size, ok := s.mallocs[start]
	if !ok {
		return InvalidAddress(start)
	}
	end := start + size - 1
	ep := s.freeList.Front()
	for e := s.freeList.Front(); e != nil; e = e.Next() {
		chunk := e.Value.(Chunk)
		chunkp := ep.Value.(Chunk)
		if end+1 == chunk.start {
			// free is adjacent to beginning of this chunk
			if ep != e {
				// We are not at the beginning of the list
				if chunkp.end+1 == start {
					// Combine previous node+free+current, delete previous
					e.Value = Chunk{start: chunkp.start, end: chunk.end}
					s.freeList.Remove(ep)
					delete(s.mallocs, start)
					return nil
				}
			}
			// Just combine free+current since there is still a hole in memory or we are at
			// the had of the list
			e.Value = Chunk{start: start, end: chunk.end}
			delete(s.mallocs, start)
			return nil
		} else if chunkp.end+1 == start {
			// Get the next node to check if can combine prev+free+next
			if e.Next() != nil {
				en := e.Next()
				chunkn := en.Value.(Chunk)
				if end+1 == chunkn.start {
					en.Value = Chunk{start: chunkp.start, end: chunkn.end}
					s.freeList.Remove(ep)
					delete(s.mallocs, start)
					return nil
				}
			}
			// free is adjacent to the previous node. Add free to ep
			ep.Value = Chunk{start: chunkp.start, end: end}
			delete(s.mallocs, start)
			return nil
		} else if end+1 < chunk.start {
			// free is before this chunk but not adjacent. Add a new node
			s.freeList.InsertBefore(Chunk{start: start, end: end}, e)
			delete(s.mallocs, start)
			return nil
		}

		// Assign ep to current node and continue
		ep = e
	}

	// We should never get here
	return UknownError("Could not put free block back on the list")
}

// MProtect changes the access protections for the pages that span the memory block
func (s *Shm) MProtect(addr int, len int, prot Prot) error {
	pages, err := s.GetPages(addr, len)
	if err != nil {
		return err
	}
	for _, page := range pages {
		s.pageAccess[page] = prot
	}
	return nil
}

// GetProtStatus gets the protection status of a page
func (s *Shm) GetProtStatus(page int) Prot {
	return s.pageAccess[page]
}

// GetProtStatusPages gets the protection status of a list of pages
func (s *Shm) GetProtStatusPages(pages []int) []Prot {
	maxPage := len(s.pageAccess) - 1
	prot := make([]Prot, len(pages))
	for i, page := range pages {
		if page > maxPage {
			prot[i] = PROT_NONE
		} else {
			prot[i] = s.pageAccess[page]
		}
	}
	return prot
}

// InstallSEGVHandler installs a SEGV handler
func (s *Shm) InstallSEGVHandler(handler SEGVHandler) error {
	s.handlers = append(s.handlers, handler)
	return nil
}

// Read a single byte at the specified address
func (s *Shm) Read(addr int) (byte, error) {

	err := s.checkBounds(addr)
	if err != nil {
		return 0, OutofBounds(addr)
	}

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	// Get the protection status of this page
	prot := s.GetProtStatus(s.GetPage(addr))

	// if the page is protected, then call the handlers until one of them
	// changes the permissions, otherwise return error
	if prot&PROT_READ != PROT_READ {
		for _, segvh := range s.handlers {
			err := segvh(addr, 1, PROT_READ)
			if err == nil {
				return s.mem[addr], nil
			}
		}
		return s.mem[addr], AccessViolation(PROT_READ)
	}

	// The page is not protected. Return the value
	return s.mem[addr], nil
}

// ReadN reads an array of bytes
func (s *Shm) ReadN(addr int, len int) ([]byte, error) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	// Get the protection status of each of the pages that
	// span this block
	pages, err := s.GetPages(addr, len)
	if err != nil {
		return nil, err
	}
	protList := s.GetProtStatusPages(pages)

	// if the page is protected, then call the handlers until one of them
	// changes the permissions. Do this for each page in the list. If one
	// of the pages is not updated then return error
	for _, prot := range protList {
		if prot&PROT_READ != PROT_READ {
			handled := false
			for _, segvh := range s.handlers {
				err := segvh(addr, len, PROT_READ)
				if err == nil {
					handled = true
					break
				}
			}
			if !handled {
				return nil, AccessViolation(PROT_READ)
			}
		}
	}

	// We can now copy and return the data
	data := make([]byte, len)
	copy(data, s.mem[addr:addr+len])
	return data, nil
}

// Write a byte at address
func (s *Shm) Write(addr int, val byte) error {

	err := s.checkBounds(addr)
	if err != nil {
		return OutofBounds(addr)
	}

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	// Get the protection status of this page
	prot := s.GetProtStatus(s.GetPage(addr))

	// if the page is protected, then call the handlers until one of them
	// changes the permissions, otherwise return error
	if prot&PROT_WRITE != PROT_WRITE {
		for _, segvh := range s.handlers {
			err := segvh(addr, 1, PROT_WRITE)
			if err == nil {
				s.mem[addr] = val
				return nil
			}
		}
		return AccessViolation(PROT_WRITE)
	}

	// The page is not protected. Return the value
	s.mem[addr] = val
	return nil

}

// WriteN writes an array of bytes starting at address
func (s *Shm) WriteN(addr int, data []byte) error {
	// Get the protection status of each of the pages that
	// span this block
	length := len(data)
	pages, err := s.GetPages(addr, length)
	if err != nil {
		return err
	}
	protList := s.GetProtStatusPages(pages)

	// if the page is protected, then call the handlers until one of them
	// changes the permissions. Do this for each page in the list. If one
	// of the pages is not updated then return error
	for _, prot := range protList {
		if prot&PROT_WRITE != PROT_WRITE {
			handled := false
			for _, segvh := range s.handlers {
				err := segvh(addr, length, PROT_WRITE)
				if err == nil {
					handled = true
					break
				}
			}
			if !handled {
				return AccessViolation(PROT_WRITE)
			}
		}
	}

	// We can now copy to memory
	copy(s.mem[addr:addr+length], data)
	return nil
}

// ReadPage reads a page. This is used by the DSM library to get a whole page
func (s *Shm) ReadPage(page int) []byte {
	start := s.pageSize * page
	data := make([]byte, s.pageSize)
	copy(data, s.mem[start:start+s.pageSize])
	return data
}

// WritePage writes  a page. This is used by the DSM library to write a page
func (s *Shm) WritePage(page int, data []byte) {
	start := s.pageSize * page
	copy(s.mem[start:start+s.pageSize], data)
	return
}

// GetSize gets the size of the Shared Memory
func (s *Shm) GetSize() int {
	return s.size
}

// GetPageSize gets the page size
func (s *Shm) GetPageSize() int {
	return s.pageSize
}

// GetPage gets the page for an address
func (s *Shm) GetPage(addr int) int {
	return addr / s.pageSize
}

// GetPages gets a list of pages that span a block of memory
func (s *Shm) GetPages(start int, len int) ([]int, error) {
	if start+len > s.size {
		return nil, OutofBounds(start + len - 1)
	}
	pstart := s.GetPage(start)
	pend := s.GetPage(start + len - 1)
	pages := make([]int, pend-pstart+1)
	for p, i := pstart, 0; p <= pend; i, p = i+1, p+1 {
		pages[i] = p
	}
	return pages, nil
}

func (s *Shm) checkBounds(addr int) error {
	if addr > s.size || addr < 0 {
		return OutofBounds(addr)
	}

	return nil
}
