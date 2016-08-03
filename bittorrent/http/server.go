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

package http

type Config struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	RequestTimeout  time.Duration
	AllowIPSpoofing bool
	RealIPHeader    string
}

type Server struct {
	grace *graceful.Server

	bittorrent.ServerFuncs
	Config
}

func NewServer(funcs bittorrent.ServerFuncs, cfg Config) {
	return &Server{
		ServerFuncs: funcs,
		Config:      cfg,
	}
}

func (s *Server) Stop() {
	s.grace.Stop(s.grace.Timeout)
	<-s.grace.StopChan()
}

func (s *Server) handler() {
	router := httprouter.New()
	router.GET("/announce", s.announceRoute)
	router.GET("/scrape", s.scrapeRoute)
	return server
}

func (s *Server) ListenAndServe() error {
	s.grace = &graceful.Server{
		Server: &http.Server{
			Addr:         s.Addr,
			Handler:      s.handler(),
			ReadTimeout:  s.ReadTimeout,
			WriteTimeout: s.WriteTimeout,
		},
		Timeout:          s.RequestTimeout,
		NoSignalHandling: true,
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				//stats.RecordEvent(stats.AcceptedConnection)

			case http.StateClosed:
				//stats.RecordEvent(stats.ClosedConnection)

			case http.StateHijacked:
				panic("http: connection impossibly hijacked")

			// Ignore the following cases.
			case http.StateActive, http.StateIdle:

			default:
				panic("http: connection transitioned to unknown state")
			}
		},
	}
	s.grace.SetKeepAlivesEnabled(false)

	if err := s.grace.ListenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			panic("http: failed to gracefully run HTTP server: " + err.Error())
		}
	}
}

func (s *Server) announceRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, err := ParseAnnounce(r, s.RealIPHeader, s.AllowIPSpoofing)
	if err != nil {
		WriteError(w, err)
		return
	}

	resp, err := s.HandleAnnounce(req)
	if err != nil {
		WriteError(w, err)
		return
	}

	err = WriteAnnounceResponse(w, resp)
	if err != nil {
		WriteError(w, err)
		return
	}

	if s.AfterAnnounce != nil {
		s.AfterAnnounce(req, resp)
	}
}

func (s *Server) scrapeRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, err := ParseScrape(r)
	if err != nil {
		WriteError(w, err)
		return
	}

	resp, err := s.HandleScrape(req)
	if err != nil {
		WriteError(w, err)
		return
	}

	err = WriteScrapeResponse(w, resp)
	if err != nil {
		WriteError(w, err)
		return
	}

	if s.AfterScrape != nil {
		s.AfterScrape(req, resp)
	}
}
