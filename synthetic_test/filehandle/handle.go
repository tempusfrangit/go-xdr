package filehandle

// Handle represents an NFS file handle (opaque to clients, structured internally)
type Handle []byte

// Some methods to make it look realistic
func (h Handle) String() string {
	return "file handle"
}

func (h Handle) IsValid() bool {
	return len(h) > 0
}