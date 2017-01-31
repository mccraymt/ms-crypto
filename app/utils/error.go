package utils

type PCResponseErrorer interface {
	Error() string
	JobNumber() string
}

type PCResponseError struct {
	// Err       error
	message   string
	jobNumber string
}

func NewPCResponseError(message string, jobNumber string) *PCResponseError {
	return &PCResponseError{message: message, jobNumber: jobNumber}
}

func (e *PCResponseError) Error() string {
	return e.message
	// return e.Error()
}

func (e *PCResponseError) JobNumber() string {
	return e.jobNumber
}
