// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"fmt"
	"github.com/mdhender/ottoweb/stores/ffs"
	"github.com/mdhender/ottoweb/stores/sqlite"
	"net"
	"os"
	"path/filepath"
)

type Options []Option
type Option func(*Server) error

func withAssets(path string) Option {
	return func(s *Server) error {
		if abspath, err := filepath.Abs(path); err != nil {
			return err
		} else if sb, err := os.Stat(abspath); err != nil {
			return err
		} else if !sb.IsDir() {
			return fmt.Errorf("%s: not a directory", abspath)
		} else {
			s.paths.assets = abspath
		}
		return nil
	}
}

func withComponents(path string) Option {
	return func(s *Server) error {
		if abspath, err := filepath.Abs(path); err != nil {
			return err
		} else if sb, err := os.Stat(abspath); err != nil {
			return err
		} else if !sb.IsDir() {
			return fmt.Errorf("%s: not a directory", path)
		} else {
			s.paths.components = abspath
		}
		return nil
	}
}

func withFS(fs *ffs.FFS) Option {
	return func(s *Server) error {
		s.stores.ffs = fs
		return nil
	}
}

func withHost(host string) Option {
	return func(s *Server) error {
		s.host = host
		s.Addr = net.JoinHostPort(s.host, s.port)
		return nil
	}
}

func withPort(port string) Option {
	return func(s *Server) error {
		s.port = port
		s.Addr = net.JoinHostPort(s.host, s.port)
		return nil
	}
}

func withStaticFileServer(useStaticFileServer bool) Option {
	return func(s *Server) error {
		s.staticFileServer = useStaticFileServer
		return nil
	}
}

func withStore(store *sqlite.DB) Option {
	return func(s *Server) error {
		s.stores.sessions = store
		s.stores.store = store
		return nil
	}
}

func withUserData(path string) Option {
	return func(s *Server) error {
		if abspath, err := filepath.Abs(path); err != nil {
			return err
		} else if sb, err := os.Stat(abspath); err != nil {
			return err
		} else if !sb.IsDir() {
			return fmt.Errorf("%s: not a directory", path)
		} else {
			s.paths.userdata = abspath
		}
		return nil
	}
}
