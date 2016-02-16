// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package errors

import "net/http"

type Error struct {
	message string
	public  bool
	status  int
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Public() bool {
	return e.public
}

func (e *Error) Status() int {
	return e.status
}

func NewBadRequest(msg string) error {
	return &Error{
		message: msg,
		public:  true,
		status:  http.StatusBadRequest,
	}
}

func NewMessage(msg string) error {
	return &Error{
		message: msg,
		public:  true,
		status:  http.StatusOK,
	}
}
