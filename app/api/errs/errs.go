package errs

import (
	"errors"
	"fmt"
)

type Error struct {
	Code    ErrCode `json:"code"`
	Message string  `json:"message"`
}

// New constructs an error based  on an app error
func New(code ErrCode, err error) Error {
	return Error{
		Code:    code,
		Message: err.Error(),
	}
}

// Newf constructs an error based on an error message
func Newf(code ErrCode, format string, v ...any) Error {
	return Error{
		Code:    code,
		Message: fmt.Sprintf(format, v...),
	}
}

// Error implemenmts the error interface
func (err Error) Error() string {
	return err.Message
}

// IsError tests the concrete error is of the Error type
func IsError(err error) bool {
	var er Error
	return errors.As(err, &er)
}

func GetError(err error) Error {
	var er Error
	if !errors.As(err, &er) {
		return Error{}
	}
	return er
}
