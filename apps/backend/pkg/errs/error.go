package errs

import "fmt"

type Error interface {
	Code() int
	Error() string
	WrapError(msg string) Error
	InnerError(msg string)
}

type err struct {
	ErrCode int
	ErrMsg  string
}

func (e *err) Code() int {
	return e.ErrCode
}

func (e *err) Error() string {
	return e.ErrMsg
}

func (e *err) WrapError(msg string) Error {
	e.ErrMsg = fmt.Sprintf("%s, <%s>", msg, e.ErrMsg)
	return e
}

func (e *err) InnerError(msg string) {
	e.ErrMsg = fmt.Sprintf("%s, <%s>", e.ErrMsg, msg)
}

func NewError(code int) Error {
	e := &err{}
	e.ErrCode = code
	e.ErrMsg = eMap[code]
	if "" == e.ErrMsg {
		e.ErrMsg = eMap[InternalErr]
	}
	return e
}

func NewErrorWithMsg(code int, msg string) Error {
	e := &err{}
	e.ErrCode = code
	e.ErrMsg = msg
	if "" == e.ErrMsg {
		if msg, ok := eMap[code]; ok {
			e.ErrMsg = msg
		} else {
			e.ErrMsg = eMap[InternalErr]
		}
	}
	return e
}

func NewErrorWithMsgF(code int, format string, a ...interface{}) Error {
	return NewErrorWithMsg(code, fmt.Sprintf(format, a...))
}

func NewSQLMsg(err error) Error {
	e := NewErrorWithMsg(DBExecErr, err.Error())
	return e
}
