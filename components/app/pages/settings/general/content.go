// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package general

import "time"

type Content_t struct {
	ClanId           string
	AccountName      string
	Roles            string
	Data             string // path to user data folder
	ScrubReports     string // true or false
	LanguageAndDates struct {
		DateFormat string
		Timezone   struct {
			Name      string
			Automatic bool
		}
		TimezoneSelect TimezoneSelect_t
	}
	XState struct {
		On          bool
		Description string
	}
}

type TimezoneSelect_t struct {
	Options []TimezoneSelectOption_t
}

type TimezoneSelectOption_t struct {
	Name     string
	Selected bool
}

func TimezoneSelectList(loc *time.Location) (list TimezoneSelect_t) {
	var wantLocation string
	if loc != nil {
		wantLocation = loc.String()
	}
	for _, loc := range []string{
		"America/Chicago",
		"America/Denver",
		"America/Los_Angeles",
		"America/New_York",
		"America/Phoenix",
		"America/Regina",
		"Asia/Kolkata",
		"Australia/Melbourne",
		"Australia/Sydney",
		"Europe/London",
		"Pacific/Auckland",
		"UTC", // must be last
	} {
		list.Options = append(list.Options, TimezoneSelectOption_t{
			Name:     loc,
			Selected: wantLocation == loc,
		})
	}
	return list
}
