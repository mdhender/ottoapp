// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package reports

type Content_t struct {
	ClanId string
	Turns  []*Turn_t
}

type Turn_t struct {
	Turn  string // year-month
	Files []File_t
}

type File_t struct {
	ClanId string
	Report string // empty if no report
	Log    string // empty if no log
	Map    string // empty if no map
}
