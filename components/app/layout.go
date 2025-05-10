// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package app

import "github.com/mdhender/ottoweb/components/app/widgets"

type Layout struct {
	Title       string
	Heading     string
	CurrentPage struct {
		Dashboard     bool
		Maps          bool
		Reports       bool
		Calendar      bool
		Settings      bool
		Documentation bool
	}
	Scripts       []string // javascript files to include in the header
	Content       any
	Footer        Footer
	Notifications widgets.NotificationPanel_t
}

type Footer struct {
	Copyright Copyright
	Version   string
	Timestamp string
}

type Copyright struct {
	Year  int
	Owner string
}
