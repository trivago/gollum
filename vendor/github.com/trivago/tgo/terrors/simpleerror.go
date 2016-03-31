package terrors

// SimpleError provides a simple type for type based error handling.
// This type can hold a message that is returned upon Error().
type SimpleError struct {
	Message string
}

func (s SimpleError) Error() string {
	return s.Message
}
