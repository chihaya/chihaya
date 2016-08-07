// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/jzelinskie/trakr/bittorrent"
)

// Hook abstracts the concept of anything that needs to interact with a
// BitTorrent client's request and response to a BitTorrent tracker.
type Hook interface {
	HandleAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) error
	HandleScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) error
}

// HookConstructor is a function used to create a new instance of a Hook.
type HookConstructor func(interface{}) (Hook, error)

var preHooks = make(map[string]HookConstructor)

// RegisterPreHook makes a HookConstructor available by the provided name.
//
// If this function is called twice with the same name or if the
// HookConstructor is nil, it panics.
func RegisterPreHook(name string, con HookConstructor) {
	if con == nil {
		panic("trakr: could not register nil HookConstructor")
	}
	if _, dup := preHooks[name]; dup {
		panic("trakr: could not register duplicate HookConstructor: " + name)
	}
	preHooks[name] = con
}

// NewPreHook creates an instance of the given PreHook by name.
func NewPreHook(name string, config interface{}) (Hook, error) {
	con, ok := preHooks[name]
	if !ok {
		return nil, fmt.Errorf("trakr: unknown PreHook %q (forgotten import?)", name)
	}
	return con(config)
}

var postHooks = make(map[string]HookConstructor)

// RegisterPostHook makes a HookConstructor available by the provided name.
//
// If this function is called twice with the same name or if the
// HookConstructor is nil, it panics.
func RegisterPostHook(name string, con HookConstructor) {
	if con == nil {
		panic("trakr: could not register nil HookConstructor")
	}
	if _, dup := postHooks[name]; dup {
		panic("trakr: could not register duplicate HookConstructor: " + name)
	}
	preHooks[name] = con
}

// NewPostHook creates an instance of the given PostHook by name.
func NewPostHook(name string, config interface{}) (Hook, error) {
	con, ok := preHooks[name]
	if !ok {
		return nil, fmt.Errorf("trakr: unknown PostHook %q (forgotten import?)", name)
	}
	return con(config)
}
