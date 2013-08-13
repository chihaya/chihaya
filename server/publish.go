// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.
package server

import (
	zmq "github.com/alecthomas/gozmq"
)

func (s *Server) publishQueue() {
	context, err := zmq.NewContext()
	if err != nil {
		panic(err)
	}
	defer context.Close()

	socket, err := context.NewSocket(zmq.PUB)
	if err != nil {
		panic(err)
	}
	defer socket.Close()

	socket.Bind(s.conf.PubAddr)

	for msg := range s.pubChan {
		socket.Send([]byte(msg), 0)
	}
}
