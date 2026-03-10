package customerr

import "errors"

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrCtxExeeded       = errors.New("timeout exeeded")
)
