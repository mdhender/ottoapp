// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package app

// FileInfo_t is the information about a file.
// It is used to display the file in the UI.
// Fields are exported as displayable strings.
type FileInfo_t struct {
	Owner string         // clan id of the owner of the file. may differ from the clan in the file.
	Name  string         // name of the file for display purposes.
	Turn  string         // turn of the file. yyyy-mm formatted.
	Clan  string         // clan id of the clan in the file.
	Kind  FileInfoKind_e // kind of file
	Date  string         // date of the file, formatted as YYYY-MM-DD in the user's timezone.
	Time  string         // time of the file, formatted as HH:MM:SS in the user's timezone.
	Route string         // route to the file. assumes the handler respects permissions.
	Path  string         // path to the file.
}

type FileInfoKind_e int

const (
	FIKReport FileInfoKind_e = iota
	FIKMap
	FIKError
	FIKLog
)

// Less returns true if the file should be sorted before the other file.
// Sorted by newest turn, then clan by id, then kind of file (reports before maps before logs).
func (fi FileInfo_t) Less(f FileInfo_t) bool {
	if fi.Turn > f.Turn {
		return true
	} else if fi.Turn == f.Turn {
		if fi.Clan < f.Clan {
			return true
		} else if fi.Clan == f.Clan {
			return fi.Kind < f.Kind
		}
	}
	return false
}

func (fi FileInfo_t) IsError() bool {
	return fi.Kind == FIKError
}

func (fi FileInfo_t) IsLog() bool {
	return fi.Kind == FIKLog
}

func (fi FileInfo_t) IsMap() bool {
	return fi.Kind == FIKMap
}

func (fi FileInfo_t) IsOwnedBy(clanId string) bool {
	return fi.Owner == clanId
}

func (fi FileInfo_t) IsReport() bool {
	return fi.Kind == FIKReport
}
