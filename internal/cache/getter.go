package cache

// Getter loads data for a key
type Getter interface {
	// Get returns the value identified by key
	Get(key string) ([]byte, error)
}

// GetterFunc implements Getter with a function
type GetterFunc func(key string) ([]byte, error)

// Get implements the Getter interface
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}
