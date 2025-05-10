// Copyright (c) 2024 Michael D Henderson. All rights reserved.

// Package main implements the ottoapp command.
package main

import (
	"bytes"
	"github.com/mdhender/semver"
	"github.com/spf13/cobra"
	"log"
)

var (
	version = semver.Version{Major: 0, Minor: 39, Patch: 0}

	argsRoot struct {
		paths struct {
			assets     string // directory containing the assets files
			components string // directory containing the component files
			data       string // path to the data files directory
			database   string // path to the database directory
		}
		server struct {
			host   string
			port   string
			static bool // if true, serve static files from the assets directory
		}
		store struct {
			operator string
			secret   string
		}
	}

	cmdRoot = &cobra.Command{
		Use:   "ottoapp",
		Short: "Root command for our application",
		Long:  `Run a web server for TribeNet maps.`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("Hello from root command\n")
		},
	}
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	//log.Printf("version: %s\n", version.String())
	//log.Printf("version: tndocx %s\n", tndocx.Version().String())

	cmdRoot.AddCommand(cmdDb)
	cmdDb.PersistentFlags().StringVar(&argsDb.paths.database, "database", "", "path to the database file")

	cmdDb.AddCommand(cmdDbInit)
	cmdDbInit.Flags().BoolVarP(&argsDb.force, "force", "f", false, "force the creation even if the database exists")
	cmdDbInit.Flags().StringVar(&argsDb.secrets.admin, "admin-password", "", "optional password for the admin user")
	cmdDbInit.Flags().StringVarP(&argsDb.paths.assets, "assets", "a", "", "path to the assets directory")
	if err := cmdDbInit.MarkFlagRequired("assets"); err != nil {
		log.Fatalf("error: assets: %v\n", err)
	}
	cmdDbInit.Flags().StringVarP(&argsDb.paths.components, "components", "t", "", "path to the components directory")
	if err := cmdDbInit.MarkFlagRequired("components"); err != nil {
		log.Fatalf("error: components: %v\n", err)
	}
	cmdDbInit.Flags().StringVarP(&argsDb.paths.data, "data", "d", "", "path to the data files directory")
	if err := cmdDbInit.MarkFlagRequired("data"); err != nil {
		log.Fatalf("error: data: %v\n", err)
	}
	cmdDbInit.Flags().StringVarP(&argsDb.secrets.signing, "secret", "s", "", "new secret for signing tokens")

	cmdDb.AddCommand(cmdDbCreate)
	cmdDbCreate.AddCommand(cmdDbCreateUser)
	cmdDbCreateUser.Flags().StringVarP(&argsDb.data.user.clan, "clan-id", "c", "", "clan number for user")
	if err := cmdDbCreateUser.MarkFlagRequired("clan-id"); err != nil {
		log.Fatalf("error: clan-id: %v\n", err)
	}
	cmdDbCreateUser.Flags().StringVarP(&argsDb.data.user.email, "email", "e", "", "email for user")
	if err := cmdDbCreateUser.MarkFlagRequired("email"); err != nil {
		log.Fatalf("error: email: %v\n", err)
	}
	cmdDbCreateUser.Flags().StringVarP(&argsDb.data.user.secret, "secret", "s", "", "secret for user")
	cmdDbCreateUser.Flags().StringVarP(&argsDb.data.user.timezone, "timezone", "t", "UTC", "timezone for user")
	cmdDbCreateUser.Flags().BoolVar(&argsDb.data.user.usePhrase, "use-phrases", false, "generate secret phrase for user")

	cmdDb.AddCommand(cmdDbDelete)
	cmdDbDelete.AddCommand(cmdDbDeleteUser)
	cmdDbDeleteUser.Flags().StringVarP(&argsDb.data.user.clan, "clan-id", "c", "", "clan number for user")
	if err := cmdDbCreateUser.MarkFlagRequired("clan-id"); err != nil {
		log.Fatalf("error: clan-id: %v\n", err)
	}

	cmdDb.AddCommand(cmdDbUpdate)
	cmdDbUpdate.Flags().BoolVar(&argsDb.secrets.useRandomSecret, "use-random-secret", false, "generate a new random secret for signing tokens")
	cmdDbUpdate.Flags().StringVar(&argsDb.secrets.admin, "admin-password", "", "update password for the admin user")
	cmdDbUpdate.Flags().StringVarP(&argsDb.paths.assets, "assets", "a", "", "new path to the assets directory")
	cmdDbUpdate.Flags().StringVarP(&argsDb.paths.data, "data", "d", "", "new path to the data files directory")
	cmdDbUpdate.Flags().StringVarP(&argsDb.paths.components, "components", "t", "", "new path to the components directory")
	cmdDbUpdate.Flags().StringVarP(&argsDb.secrets.signing, "secret", "s", "", "new secret for signing tokens")
	cmdDbUpdate.AddCommand(cmdDbUpdateUser)
	cmdDbUpdateUser.AddCommand(cmdDbUpdateUserPassword)
	cmdDbUpdateUserPassword.Flags().StringVarP(&argsDb.data.user.clan, "clan-id", "c", "", "clan id to update")
	if err := cmdDbUpdateUserPassword.MarkFlagRequired("clan-id"); err != nil {
		log.Fatalf("error: clan-id: %v\n", err)
	}
	cmdDbUpdateUserPassword.Flags().BoolVar(&argsDb.data.user.isActive, "is-active", true, "active clan")
	cmdDbUpdateUser.AddCommand(cmdDbUpdateUserTimezone)
	cmdDbUpdateUserTimezone.Flags().StringVarP(&argsDb.data.user.clan, "clan-id", "c", "", "clan id to update")
	if err := cmdDbUpdateUserTimezone.MarkFlagRequired("clan-id"); err != nil {
		log.Fatalf("error: clan-id: %v\n", err)
	}
	cmdDbUpdateUserTimezone.Flags().StringVarP(&argsDb.data.user.timezone, "timezone", "t", "UTC", "timezone for user")
	if err := cmdDbUpdateUserTimezone.MarkFlagRequired("timezone"); err != nil {
		log.Fatalf("error: timezone: %v\n", err)
	}

	cmdRoot.AddCommand(cmdServe)
	cmdServe.Flags().StringVarP(&argsServe.paths.database, "database", "d", "", "path to database file")
	if err := cmdServe.MarkFlagRequired("database"); err != nil {
		log.Fatalf("error: database: %v\n", err)
	}
	cmdServe.Flags().StringVar(&argsServe.server.host, "host", "localhost", "host to serve on")
	cmdServe.Flags().StringVar(&argsServe.server.port, "port", "29631", "port to bind to")
	cmdServe.Flags().BoolVar(&argsServe.server.static, "serve-static-files", true, "serve static files from the assets directory")

	cmdRoot.AddCommand(cmdVersion)

	if err := cmdRoot.Execute(); err != nil {
		log.Fatal(err)
	}
}

func cbIsSet(s string) bool {
	return s == "on" || s == "true"
}

// return true if a line is blank (empty).
func isBlankLine(line []byte) bool {
	return len(line) == 0
}

// replaceInvalidUTF8 replaces all invalid UTF-8 sequences in a byte slice with spaces.
func replaceInvalidUTF8(data []byte) []byte {
	// no op until we figure out how to handle invalid UTF-8 sequences
	return data
	//var result []byte
	//for len(data) > 0 {
	//	if r, size := utf8.DecodeRune(data); r == utf8.RuneError && size == 1 {
	//		// Invalid UTF-8 byte found, replace with space.
	//		result = append(result, ' ')
	//		data = data[1:] // Move to the next byte.
	//	} else {
	//		// Valid rune, copy it to the result.
	//		result = append(result, data[:size]...)
	//		data = data[size:] // Move past the valid rune.
	//	}
	//}
	//return result
}

// trim leading blank lines from the slice of byte slices
func trimLeadingBlankLines(lines [][]byte) [][]byte {
	for len(lines) != 0 && isBlankLine(lines[0]) {
		lines = lines[1:]
	}
	return lines
}

func trimNonMappingLines(lines [][]byte) [][]byte {
	for n, line := range lines {
		if bytes.HasPrefix(line, []byte("Current Turn ")) {
			continue
		} else if bytes.HasPrefix(line, []byte("Tribe Movement: ")) {
			continue
		} else if bytes.HasPrefix(line, []byte("Scout ")) {
			continue
		} else if bytes.Contains(line, []byte("(Previous Hex =")) {
			continue
		} else if bytes.Contains(line, []byte(" Fleet Movement:")) {
			continue
		} else if bytes.Contains(line, []byte(" Status:")) {
			continue
		}
		lines[n] = []byte{}
	}
	return lines
}

// trim trailing blank lines from the slice of byte slices
func trimTrailingBlankLines(lines [][]byte) [][]byte {
	end := len(lines)
	for end > 0 && isBlankLine(lines[end-1]) {
		end--
	}
	return lines[:end]
}
