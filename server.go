// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"fmt"
	"github.com/mdhender/ottapp/components/app"
	"github.com/mdhender/ottapp/domains"
	"github.com/mdhender/ottapp/stores/ffs"
	"github.com/mdhender/ottapp/stores/sqlite"
	"log"
	"net"
	"net/http"
	"time"
)

func newServer(options ...Option) (*Server, error) {
	s := &Server{
		scheme: "http",
		mux:    http.NewServeMux(),
		blocks: struct {
			Footer app.Footer
		}{
			Footer: app.Footer{
				Copyright: app.Copyright{
					Year:  2024,
					Owner: "Michael D Henderson",
				},
				Version: version.String(),
			},
		},
	}
	s.Addr = net.JoinHostPort(s.host, s.port)
	s.MaxHeaderBytes = 1 << 20
	s.IdleTimeout = 10 * time.Second
	s.ReadTimeout = 5 * time.Second
	s.WriteTimeout = 10 * time.Second

	s.sessions.cookieName = "ottoapp"
	s.sessions.rememberMe = "ottoapp1-clan-idff-b364-a70ced220fff"
	s.sessions.ttl = 2 * 7 * 24 * time.Hour
	s.sessions.maxAge = 2 * 7 * 24 * 60 * 60 // 2 weeks

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, err
		}
	}

	// get the path to the user data from the database
	assets, components, userdata, err := s.stores.store.GetServerPaths()
	if err != nil {
		return nil, err
	}
	if err := withAssets(assets)(s); err != nil {
		return nil, err
	}
	if err := withComponents(components)(s); err != nil {
		return nil, err
	}
	if err := withUserData(userdata)(s); err != nil {
		return nil, err
	}

	if fs, err := ffs.New(s.paths.userdata); err != nil {
		log.Fatalf("error: %v\n", err)
	} else if err = withFS(fs)(s); err != nil {
		return nil, err
	}

	s.mux = s.routes()

	return s, nil
}

type Server struct {
	http.Server
	scheme, host, port string
	mux                *http.ServeMux
	staticFileServer   bool
	stores             struct {
		ffs      *ffs.FFS
		sessions *sqlite.DB
		store    *sqlite.DB
	}
	//assets             fs.FS
	//components          fs.FS
	paths struct {
		assets     string
		components string
		userdata   string
	}
	sessions struct {
		cookieName string
		maxAge     int // maximum age of a session cookie in seconds
		rememberMe string
		ttl        time.Duration
	}
	blocks struct {
		Footer app.Footer
	}
	features struct {
		cacheBuster bool
	}
}

func (s *Server) BaseURL() string {
	return fmt.Sprintf("%s://%s", s.scheme, s.Addr)
}

// extractSession extracts the session from the request.
// Returns nil if there is no session, or it is invalid.
func (s *Server) extractSession(r *http.Request) (*domains.User_t, error) {
	cookie, err := r.Cookie(s.sessions.cookieName)
	if err != nil {
		return nil, nil
	}

	user, err := s.stores.sessions.GetSession(cookie.Value)
	if err != nil {
		return nil, err
	}

	return user, nil
}
