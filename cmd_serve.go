// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/mdhender/ottoweb/stores/sqlite"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var (
	argsServe struct {
		paths struct {
			database string // path to the database file
		}
		server struct {
			host   string
			port   string
			static bool // if true, serve static files from the assets directory
		}
	}

	cmdServe = &cobra.Command{
		Use:   "serve",
		Short: "serve the web application",
		Long:  `Serve the web application.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if argsServe.paths.database == "" {
				return fmt.Errorf("error: assets: path is required\n")
			} else if path, err := filepath.Abs(argsServe.paths.database); err != nil {
				return fmt.Errorf("database: %v\n", err)
			} else if ok, err := isfile(path); err != nil {
				return fmt.Errorf("database: %v\n", err)
			} else if !ok {
				return fmt.Errorf("database: %s: not a file\n", path)
			} else {
				argsServe.paths.database = path
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			started := time.Now()

			log.Printf("host      : %s\n", argsServe.server.host)
			log.Printf("port      : %s\n", argsServe.server.port)
			log.Printf("database  : %s\n", argsServe.paths.database)
			log.Printf("staticfs  : %v\n", argsServe.server.static)

			// open the database
			log.Printf("database : %s\n", argsServe.paths.database)
			ctx := context.Background()
			store, err := sqlite.Open(argsServe.paths.database, ctx)
			if err != nil {
				log.Fatalf("error: store: %v\n", err)
			}
			defer func() {
				if store != nil {
					_ = store.Close()
				}
				store = nil
			}()

			s, err := newServer(
				withHost(argsServe.server.host),
				withPort(argsServe.server.port),
				withStaticFileServer(argsServe.server.static),
				withStore(store),
			)
			if err != nil {
				log.Fatalf("error: %v\n", err)
			}

			// create a channel to listen for OS signals.
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

			// start the server in a goroutine so that it doesn't block.
			go func() {
				log.Printf("listening on %s\n", s.BaseURL())
				if err := http.ListenAndServe(s.Addr, s.mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Printf("server: %v\n", err)
				}
				log.Printf("server: shutdown\n")
			}()

			// server is running; block until we receive a signal.
			sig := <-stop

			log.Printf("signal: received %v (%v)\n", sig, time.Since(started))

			// close the database connection
			// todo: db close may wait on pending transactions. will this cause a race condition?
			log.Printf("closing store (%v)\n", time.Since(started))
			if err := store.Close(); err != nil {
				log.Printf("store: close %v\n", err)
			}
			log.Printf("store: closed (%v)\n", time.Since(started))

			// graceful shutdown with a timeout.
			timeout := time.Second * 5
			log.Printf("creating context with %v timeout (%v)\n", timeout, time.Since(started))
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// cancel any idle connections.
			log.Printf("canceling idle connections (%v)\n", time.Since(started))
			s.SetKeepAlivesEnabled(false)

			log.Printf("sending signal to shut down the server (%v)\n", time.Since(started))
			if err := s.Shutdown(ctx); err != nil {
				log.Fatalf("server: shutdown: %v\n", err)
			}

			log.Printf("server stopped ¡gracefully! (%v)\n", time.Since(started))
		},
	}
)
