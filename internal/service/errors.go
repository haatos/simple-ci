package service

type ErrRunQueueFull struct{}

func (e ErrRunQueueFull) Error() string {
	return "run queue is full"
}

func NewErrRunQueueFull() *ErrRunQueueFull {
	return &ErrRunQueueFull{}
}

type RunCancelError struct {
	Message string
}

func (rce RunCancelError) Error() string {
	return rce.Message
}
