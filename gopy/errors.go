package gopy

import "fmt"

type PythonError struct {
	Type      string `msgpack:"type,omitempty"`
	Message   string `msgpack:"message,omitempty"`
	Traceback string `msgpack:"traceback,omitempty"`
}

func (e *PythonError) Error() string {
	if e == nil {
		return "python error"
	}
	if e.Type != "" && e.Message != "" {
		return fmt.Sprintf("python %s: %s", e.Type, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("python error: %s", e.Message)
	}
	if e.Type != "" {
		return fmt.Sprintf("python %s", e.Type)
	}
	return "python error"
}
