package repository

// InputError marks a validation failure whose message is safe to show to API
// clients. Any other repository error is internal and must not be echoed.
type InputError struct{ Message string }

func (e *InputError) Error() string { return e.Message }

func invalidInput(message string) error { return &InputError{Message: message} }

// asInputError wraps a validation error from another package (recurrence).
func asInputError(err error) error {
	if err == nil {
		return nil
	}
	return &InputError{Message: err.Error()}
}
