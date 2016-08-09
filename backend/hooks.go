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
	"golang.org/x/net/context"

	"github.com/jzelinskie/trakr/bittorrent"
)

// Hook abstracts the concept of anything that needs to interact with a
// BitTorrent client's request and response to a BitTorrent tracker.
type Hook interface {
	HandleAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) error
	HandleScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) error
}

type nopHook struct{}

func (nopHook) HandleAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) error {
	return nil
}

func (nopHook) HandleScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) error {
	return nil
}
