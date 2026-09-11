// inheritance.go — embedding

package app

import "fmt"

type Base struct{}

type Printer struct{}

type Reader interface {
	Read() error
}

type Writer interface {
	Write() error
}

type Ledger struct {
	Base                // inheritance 1
	*Printer            // inheritance 1 — a pointer embedding still embeds
	fmt.Stringer        // inheritance 1, stdlib_coupling 1 (fmt)
	name         string // local_variable 1
}

type ReadWriter interface {
	Reader // inheritance 1
	Writer // inheritance 1
	Close() error
}

type Number interface {
	~int | ~float64 // inheritance 0 — type terms and unions are not embedding
}

type Stringish interface {
	fmt.Stringer // inheritance 1, stdlib_coupling 1 (fmt)
	~string      // inheritance 0
}

// unit Ledger: struct, inheritance 3, local_variable 1, stdlib_coupling 1
// unit ReadWriter: interface, inheritance 2
// unit Number: interface, inheritance 0
// unit Stringish: interface, inheritance 1, stdlib_coupling 1
// units Base, Printer, Reader, Writer: every metric 0
