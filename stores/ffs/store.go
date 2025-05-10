// Copyright (c) 2024 Michael D Henderson. All rights reserved.

// Package ffs implements a file-based flat file system.
package ffs

import (
	"bytes"
	"fmt"
	"github.com/mdhender/ottoapp/domains"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// New assumes that files are kept in a directory structure like this:
//
//	ClanId is the in the root directory
//	ClanId/data is a legacy directory and is needed for ottomap to work
//	ClanId/data/input contains the input files (turn reports) for the clan
//	ClanId/data/output contains the output files (turn maps and error logs) for the clan
//
// The path variable points to the root directory.
func New(path string) (*FFS, error) {
	return &FFS{
		path:          path,
		rxLogFail:     regexp.MustCompile(`^([0-9]{4})-([0-9]{2})\.([0-9]{4})\.err`),
		rxLogPass:     regexp.MustCompile(`^([0-9]{4})-([0-9]{2})\.([0-9]{4})\.log`),
		rxTurnMap:     regexp.MustCompile(`^([0-9]{4})-([0-9]{2})\.([0-9]{4})\.wxx`),
		rxTurnReports: regexp.MustCompile(`^([0-9]{4})-([0-9]{2})\.([0-9]{4})\.report\.txt`),
	}, nil
}

type FFS struct {
	path          string
	rxLogFail     *regexp.Regexp
	rxLogPass     *regexp.Regexp
	rxTurnMap     *regexp.Regexp
	rxTurnReports *regexp.Regexp
}

// GetClans returns a list of all the clans in the file system.
// Intended for the administration page.
func (f *FFS) GetClans(id string) ([]string, error) {
	var clans []string

	entries, err := os.ReadDir(filepath.Join(f.path, id))
	if err != nil {
		log.Printf("ffs: getClans: %v\n", err)
		return nil, nil
	}

	// find all turn reports and add them to the list of clans
	list := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := f.rxTurnReports.FindStringSubmatch(entry.Name())
		if len(matches) != 3 {
			continue
		}
		clan := matches[2]
		list[clan] = true
	}

	for k := range list {
		clans = append(clans, k)
	}

	// sort the list, not sure why.
	sort.Strings(clans)

	return clans, nil
}

type ClanFiles_t struct {
	Errors      string // path to clan's error directory
	ErrorFiles  []File_t
	Logs        string // path to clan's log directory
	LogFiles    []File_t
	Maps        string // path to clan's output directory
	MapFiles    []File_t
	Reports     string // path to clan's input directory
	ReportFiles []File_t
}

type File_t struct {
	Name      string // file name
	Turn      string // year-month
	Year      int
	Month     int
	Clan      string
	Path      string    // full path to file
	Timestamp time.Time // must be UTC
}

func (f *FFS) GetClanFiles(user *domains.User_t) (ClanFiles_t, error) {
	if user == nil {
		return ClanFiles_t{}, nil
	}
	log.Printf("gcf: clan %q: %s\n", user.Clan, user.Data)

	files := ClanFiles_t{
		Errors:  filepath.Join(user.Data, "logs"),
		Logs:    filepath.Join(user.Data, "logs"),
		Maps:    filepath.Join(user.Data, "output"),
		Reports: filepath.Join(user.Data, "input"),
	}
	clanId := user.Clan

	// find all turn reports and add them to the list of clans
	if entries, err := os.ReadDir(files.Reports); err != nil {
		log.Printf("ffs: getClanFiles: %q: %v\n", clanId, err)
		return files, err
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if m := f.rxTurnReports.FindStringSubmatch(entry.Name()); len(m) == 4 {
				//log.Printf("ffs: getClanFiles: m %v\n", m)
				year, month, clan := m[1], m[2], m[3]
				ft := File_t{
					Name: entry.Name(),
					Clan: clan,
					Path: filepath.Join(files.Reports, entry.Name()),
				}
				if ft.Year, err = strconv.Atoi(year); err != nil {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, err
				} else if ft.Year < 899 || ft.Year > 999 {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, domains.ErrInvalidTurnYear
				} else if ft.Month, err = strconv.Atoi(month); err != nil {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, err
				} else if ft.Month < 1 || ft.Month > 12 {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, domains.ErrInvalidTurnMonth
				} else if n, err := strconv.Atoi(clan); err != nil {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, err
				} else if n < 1 || n > 999 {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, domains.ErrInvalidClan
				} else if fi, err := entry.Info(); err != nil {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, err
				} else {
					ft.Timestamp = fi.ModTime().UTC()
				}
				ft.Turn = fmt.Sprintf("%04d-%02d", ft.Year, ft.Month)
				files.ReportFiles = append(files.ReportFiles, ft)
			}
		}
	}

	// find all map files and add them to the list of clans
	if entries, err := os.ReadDir(files.Maps); err != nil {
		log.Printf("ffs: getClanFiles: %q: %v\n", clanId, err)
		return files, err
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if m := f.rxTurnMap.FindStringSubmatch(entry.Name()); len(m) == 4 {
				//log.Printf("ffs: getClanFiles: m %v\n", m)
				year, month, clan := m[1], m[2], m[3]
				ft := File_t{
					Name: entry.Name(),
					Clan: clan,
					Path: filepath.Join(files.Maps, entry.Name()),
				}
				if ft.Year, err = strconv.Atoi(year); err != nil {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, err
				} else if ft.Year < 899 || ft.Year > 999 {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, domains.ErrInvalidTurnYear
				} else if ft.Month, err = strconv.Atoi(month); err != nil {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, err
				} else if ft.Month < 1 || ft.Month > 12 {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, domains.ErrInvalidTurnMonth
				} else if n, err := strconv.Atoi(clan); err != nil {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, err
				} else if n < 1 || n > 999 {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, domains.ErrInvalidClan
				} else if fi, err := entry.Info(); err != nil {
					//log.Printf("ffs: getClanFiles: %v\n", err)
					return files, err
				} else {
					ft.Timestamp = fi.ModTime().UTC()
				}
				ft.Turn = fmt.Sprintf("%04d-%02d", ft.Year, ft.Month)
				files.MapFiles = append(files.MapFiles, ft)
			}
		}
	}

	// find all error files and add them to the list of clans
	if entries, err := os.ReadDir(files.Errors); err != nil {
		log.Printf("ffs: getClanFiles: %q: %v\n", clanId, err)
		return files, err
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			m := f.rxLogFail.FindStringSubmatch(entry.Name())
			if len(m) != 4 {
				continue
			}
			//log.Printf("ffs: getClanFiles: m %v\n", m)
			year, month, clan := m[1], m[2], m[3]
			ft := File_t{
				Name: entry.Name(),
				Clan: clan,
				Path: filepath.Join(files.Errors, entry.Name()),
			}
			if ft.Year, err = strconv.Atoi(year); err != nil {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, err
			} else if ft.Year < 899 || ft.Year > 999 {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, domains.ErrInvalidTurnYear
			} else if ft.Month, err = strconv.Atoi(month); err != nil {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, err
			} else if ft.Month < 1 || ft.Month > 12 {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, domains.ErrInvalidTurnMonth
			} else if n, err := strconv.Atoi(clan); err != nil {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, err
			} else if n < 1 || n > 999 {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, domains.ErrInvalidClan
			} else if fi, err := entry.Info(); err != nil {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, err
			} else {
				ft.Timestamp = fi.ModTime().UTC()
			}
			ft.Turn = fmt.Sprintf("%04d-%02d", ft.Year, ft.Month)
			files.ErrorFiles = append(files.ErrorFiles, ft)
		}
	}

	// find all log files and add them to the list of clans
	if entries, err := os.ReadDir(files.Logs); err != nil {
		log.Printf("ffs: getClanFiles: %q: %v\n", clanId, err)
		return files, err
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			m := f.rxLogPass.FindStringSubmatch(entry.Name())
			if len(m) != 4 {
				continue
			}
			//log.Printf("ffs: getClanFiles: m %v\n", m)
			year, month, clan := m[1], m[2], m[3]
			ft := File_t{
				Name: entry.Name(),
				Clan: clan,
				Path: filepath.Join(files.Logs, entry.Name()),
			}
			if ft.Year, err = strconv.Atoi(year); err != nil {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, err
			} else if ft.Year < 899 || ft.Year > 999 {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, domains.ErrInvalidTurnYear
			} else if ft.Month, err = strconv.Atoi(month); err != nil {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, err
			} else if ft.Month < 1 || ft.Month > 12 {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, domains.ErrInvalidTurnMonth
			} else if n, err := strconv.Atoi(clan); err != nil {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, err
			} else if n < 1 || n > 999 {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, domains.ErrInvalidClan
			} else if fi, err := entry.Info(); err != nil {
				//log.Printf("ffs: getClanFiles: %v\n", err)
				return files, err
			} else {
				ft.Timestamp = fi.ModTime().UTC()
			}
			ft.Turn = fmt.Sprintf("%04d-%02d", ft.Year, ft.Month)
			files.LogFiles = append(files.LogFiles, ft)
		}
	}

	sort.Slice(files.ErrorFiles, func(i, j int) bool {
		a, b := files.ErrorFiles[i], files.ErrorFiles[j]
		if a.Year < b.Year {
			return true
		} else if a.Year == b.Year {
			if a.Month < b.Month {
				return true
			} else if a.Month == b.Month {
				if a.Clan < b.Clan {
					return true
				} else if a.Clan == b.Clan {
					return a.Name < b.Name
				}
			}
		}
		return false
	})
	sort.Slice(files.LogFiles, func(i, j int) bool {
		a, b := files.LogFiles[i], files.LogFiles[j]
		if a.Year < b.Year {
			return true
		} else if a.Year == b.Year {
			if a.Month < b.Month {
				return true
			} else if a.Month == b.Month {
				if a.Clan < b.Clan {
					return true
				} else if a.Clan == b.Clan {
					return a.Name < b.Name
				}
			}
		}
		return false
	})
	sort.Slice(files.MapFiles, func(i, j int) bool {
		a, b := files.MapFiles[i], files.MapFiles[j]
		if a.Year < b.Year {
			return true
		} else if a.Year == b.Year {
			if a.Month < b.Month {
				return true
			} else if a.Month == b.Month {
				return a.Clan < b.Clan
			}
		}
		return false
	})

	sort.Slice(files.ReportFiles, func(i, j int) bool {
		a, b := files.ReportFiles[i], files.ReportFiles[j]
		if a.Year < b.Year {
			return true
		} else if a.Year == b.Year {
			if a.Month < b.Month {
				return true
			} else if a.Month == b.Month {
				return a.Clan < b.Clan
			}
		}
		return false
	})

	return files, nil
}

type Turn_t struct {
	Id string
}

// GetTurnListing scan the data path for turn reports and adds them to the list
func (f *FFS) GetTurnListing(id string) (list []Turn_t, err error) {
	entries, err := os.ReadDir(filepath.Join(f.path, id))
	if err != nil {
		log.Printf("ffs: getTurnListing: %v\n", err)
		return nil, nil
	}

	// add all turn reports to the list
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := f.rxTurnReports.FindStringSubmatch(entry.Name())
		if len(matches) != 3 {
			continue
		}
		list = append(list, Turn_t{Id: matches[1]})
	}

	// sort the list, not sure why.
	sort.Slice(list, func(i, j int) bool {
		return list[i].Id < list[j].Id
	})

	return list, nil
}

type TurnDetail_t struct {
	Id    string
	Clans []string
	Maps  []string
}

func (f *FFS) GetTurnDetails(id string, turnId string) (row TurnDetail_t, err error) {
	entries, err := os.ReadDir(filepath.Join(f.path, id))
	if err != nil {
		log.Printf("ffs: getTurnDetails: %v\n", err)
		return row, nil
	}

	row.Id = turnId

	// find all turn reports for this turn and collect the clan names.
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if matches := f.rxTurnReports.FindStringSubmatch(entry.Name()); len(matches) == 3 && matches[1] == turnId {
			row.Clans = append(row.Clans, matches[2])
		} else if matches = f.rxTurnMap.FindStringSubmatch(entry.Name()); len(matches) == 3 && matches[1] == turnId {
			row.Maps = append(row.Maps, matches[0])
		}
	}

	// sort the list, not sure why.
	sort.Slice(row.Clans, func(i, j int) bool {
		return row.Clans[i] < row.Clans[j]
	})

	return row, nil
}

type TurnReportDetails_t struct {
	Id    string
	Clan  string
	Map   string // set only if there is a single map
	Units []UnitDetails_t
}

type UnitDetails_t struct {
	Id          string
	CurrentHex  string
	PreviousHex string
}

func (f *FFS) GetTurnReportDetails(id string, turnId, clanId string) (report TurnReportDetails_t, err error) {
	rxCourierSection := regexp.MustCompile(`^Courier (\d{4}c)\d, `)
	rxElementSection := regexp.MustCompile(`^Element (\d{4}e)\d, `)
	rxFleetSection := regexp.MustCompile(`^Fleet (\d{4}f)\d, `)
	rxGarrisonSection := regexp.MustCompile(`^Garrison (\d{4}g)\d, `)
	rxTribeSection := regexp.MustCompile(`^Tribe (\d{4}), `)

	mapFileName := fmt.Sprintf("%s.%s.wxx", turnId, clanId)
	if sb, err := os.Stat(filepath.Join(f.path, id, mapFileName)); err == nil && sb.Mode().IsRegular() {
		report.Map = mapFileName
	}

	turnReportFile := filepath.Join(f.path, id, fmt.Sprintf("%s.%s.report.txt", turnId, clanId))
	if data, err := os.ReadFile(turnReportFile); err != nil {
		log.Printf("getTurnSections: %s: %v\n", turnReportFile, err)
	} else {
		for _, line := range bytes.Split(data, []byte("\n")) {
			if matches := rxCourierSection.FindStringSubmatch(string(line)); len(matches) == 2 {
				report.Units = append(report.Units, UnitDetails_t{
					Id: matches[1],
				})
			} else if matches = rxElementSection.FindStringSubmatch(string(line)); len(matches) == 2 {
				report.Units = append(report.Units, UnitDetails_t{
					Id: matches[1],
				})
			} else if matches = rxFleetSection.FindStringSubmatch(string(line)); len(matches) == 2 {
				report.Units = append(report.Units, UnitDetails_t{
					Id: matches[1],
				})
			} else if matches = rxGarrisonSection.FindStringSubmatch(string(line)); len(matches) == 2 {
				report.Units = append(report.Units, UnitDetails_t{
					Id: matches[1],
				})
			} else if matches = rxTribeSection.FindStringSubmatch(string(line)); len(matches) == 2 {
				report.Units = append(report.Units, UnitDetails_t{
					Id: matches[1],
				})
			}
		}
	}

	// sort the list, not sure why.
	sort.Slice(report.Units, func(i, j int) bool {
		return report.Units[i].Id < report.Units[j].Id
	})

	return report, nil
}
