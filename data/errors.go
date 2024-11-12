package data

import "fmt"

// temporary is an error interface that implements the Temporary method. It
// indicates an error is "temporary" and thus the operations being performed
// that encountered it can be retried
type temporary interface {
	Temporary() bool
}

// IsTemporary is a helper function that returns true if the arg 'err' implements
// the 'temporary' interface
func IsTemporary(err error) bool {
	te, ok := err.(temporary)
	return ok && te.Temporary()
}

type JsonMarshalError struct {
	OriginalErr error
	Data        []byte
}

func (e JsonMarshalError) Error() string {
	return fmt.Sprintf("failed marshalling json; likely unsupported data type | %v", e.OriginalErr)
}

type ReadWriteFileError struct {
	OriginalErr error
}

func (e ReadWriteFileError) Error() string {
	return fmt.Sprintf("failed to open, read, write to, or close a file | %v", e.OriginalErr)
}

func (e ReadWriteFileError) Temporary() bool {
	return true
}
