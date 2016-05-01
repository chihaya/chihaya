// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"fmt"

	"github.com/julienschmidt/httprouter"
	"github.com/tylerb/graceful"
)

// ErrInvalidMethod is returned for attempts to register or activate an endpoint
// with an invalid HTTP method.
var ErrInvalidMethod = errors.New("invalid method")

// ErrInvalidHandler is returned for attempts to register a nil handler.
var ErrInvalidHandler = errors.New("invalid handler")

// ErrRouteNotRegistered is returned for attempts to activate a route without a
// previously registered handler.
var ErrRouteNotRegistered = errors.New("route not registered")

var registeredRoutes map[string]map[string]ResponseFunc
var activatedRoutes map[string]map[string]ResponseFunc

// RegisterHandler registers the given ResponseFunc at the given path and
// HTTP method.
//
// Calling this function twice with the same method and path will overwrite
// the ResponseFunc registered earlier.
// Calling this function with a nil ResponseFunc will return ErrInvalidHandler.
// Calling this function with an invalid method returns ErrInvalidMethod.
func RegisterHandler(method, path string, handler ResponseFunc) error {
	if handler == nil {
		return ErrInvalidHandler
	}

	m := strings.ToUpper(method)
	if m != http.MethodGet && m != http.MethodPut && m != http.MethodDelete {
		return ErrInvalidMethod
	}

	if registeredRoutes[m] == nil {
		registeredRoutes[m] = make(map[string]ResponseFunc)
	}

	registeredRoutes[m][path] = handler

	log.Printf("Store API: Registered: %s %s", method, path)

	return nil
}

// RegisterNoResponseHandler registers the given NoResponseHandler at the given
// path and HTTP method.
//
// Internally, this function encapsulates the given NoResultResponseFunc in a
// ResponseFunc and calls RegisterHandler.
//
// Calling this function twice with the same method and path will overwrite
// the ResponseFunc registered earlier.
// Calling this function with a nil NoResultResponseFunc will return
// ErrInvalidHandler.
// Calling this function with an invalid method returns ErrInvalidMethod.
func RegisterNoResponseHandler(method, path string, handler NoResultResponseFunc) error {
	if handler == nil {
		return ErrInvalidHandler
	}

	return RegisterHandler(method, path, noResultHandler(handler))
}

// ActivateRoute marks the given method/path combination active and enables the
// handler previously registered for that endpoint.
//
// This function should be called by a MiddlewareConstructor once while the
// tracker boots.
//
// Calling this function with an invalid method returns ErrInvalidMethod.
// Calling this function for an endpoint without a previously registered handler
// will return ErrRouteNotRegistered.
// Activating the exact same route twice will panic.
// Activating a route with the same behaviour but different representations
// will panic in routes().
func ActivateRoute(method, path string) error {
	m := strings.ToUpper(method)
	if m != http.MethodGet && m != http.MethodPut && m != http.MethodDelete {
		return ErrInvalidMethod
	}

	if registeredRoutes[m] == nil || registeredRoutes[m][path] == nil {
		return ErrRouteNotRegistered
	}

	if activatedRoutes[m] == nil {
		activatedRoutes[m] = make(map[string]ResponseFunc)
	}

	if activatedRoutes[m][path] != nil {
		panic(fmt.Sprintf("route %s %s activated more than once", m, path))
		return nil
	}

	activatedRoutes[m][path] = registeredRoutes[m][path]

	log.Printf("Store API: Activated: %s %s", method, path)

	return nil
}

func (s *Store) routes() http.Handler {
	r := httprouter.New()
	r.GET("/ips/:ip", makeHandler(s.handleGetIP))
	r.PUT("/ips/:ip", makeHandler(noResultHandler(s.handlePutIP)))
	r.DELETE("/ips/:ip", makeHandler(noResultHandler(s.handleDeleteIP)))

	r.PUT("/networks/*network", makeHandler(noResultHandler(s.handlePutNetwork)))
	r.DELETE("/networks/*network", makeHandler(noResultHandler(s.handleDeleteNetwork)))

	r.GET("/strings/*string", makeHandler(s.handleGetString))
	r.PUT("/strings/*string", makeHandler(noResultHandler(s.handlePutString)))
	r.DELETE("/strings/*string", makeHandler(noResultHandler(s.handleDeleteString)))

	// TODO(mrd0ll4r): add peerStore endpoints

	for m, paths := range activatedRoutes {
		for path, handle := range paths {
			r.Handle(m, path, makeHandler(handle))
		}
	}

	return r
}

// Start starts the store drivers and blocks until all of them exit.
func (s *Store) Start() {
	s.grace = &graceful.Server{
		Server: &http.Server{
			Addr:         s.cfg.Addr,
			Handler:      s.routes(),
			ReadTimeout:  s.cfg.ReadTimeout,
			WriteTimeout: s.cfg.WriteTimeout,
		},
		Timeout:          s.cfg.RequestTimeout,
		NoSignalHandling: true,
	}

	if err := s.grace.ListenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			log.Printf("Failed to gracefully run store server: %s", err.Error())
			panic(err)
		}
	}
	log.Println("Store server shut down cleanly")

	<-s.shutdown
	s.wg.Wait()
	log.Println("Store shut down cleanly")
}

// Stop stops the store drivers and waits for them to exit.
func (s *Store) Stop() {
	s.grace.Stop(s.grace.Timeout)
	<-s.grace.StopChan()

	close(s.shutdown)
	s.wg.Wait()
}
