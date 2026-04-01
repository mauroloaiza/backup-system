package destination

import "io"

// Writer is the interface that all backup destinations must implement.
// Each destination receives a stream of bytes (already compressed+encrypted)
// and is responsible for persisting it to its backing store.
type Writer interface {
	// Write opens a named object in the destination and returns a WriteCloser.
	// The name is a relative path within the backup job's namespace.
	// Caller must call Close() to finalise the write.
	Write(name string) (io.WriteCloser, error)

	// Reader opens a named object for reading (restore path).
	Read(name string) (io.ReadCloser, error)

	// Delete removes a named object. Used for retention cleanup.
	Delete(name string) error

	// List returns all object names under the given prefix.
	List(prefix string) ([]string, error)

	// Close releases any persistent connections held by the destination.
	Close() error
}
