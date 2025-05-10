// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package plans

var Content = Content_t{
	Cards: []Card_t{
		{Icon: Documentation.String(),
			Title: "Documentation",
			Text:  "Better documentation on the web site.",
		},
		{Icon: People.String(),
			Title: "Implement a better onboarding process",
			Text:  "Onboard new users with a better onboarding experience.",
		},
		{Icon: Review.String(),
			Title: "Monitor Discord for support requests",
			Text:  "Check the #mapping-tools channel for any new updates.",
		},
		{Icon: Server.String(),
			Title: "Hosting",
			Text:  "Monitor load on the new server to see if it needs to be upgraded.",
		},
		{Icon: Backlog.String(),
			Title: "Update job scheduler",
			Text:  "Replace the current job scheduler with a process that runs in the server after each upload.",
		},
		{Icon: Done.String(),
			Title: "Add timestamp to footer",
			Text:  "Add a timestamp to the footer of each application page.\nDisplay the timestamp in the local time zone of the user.",
		},
		{Icon: Done.String(),
			Title: "Allow users to update their timezone",
			Text:  "Update profile to let users select and update their timezone. (Note: the footer stills needs to be updated.)",
		},
	},
}

type Content_t struct {
	Cards []Card_t
}

type Card_t struct {
	Id          int
	Icon        string
	Title       string
	Text        string
	TopRow      bool
	LeftColumn  bool
	RightColumn bool
	BottomRow   bool
}

type Icon_e int

const (
	None Icon_e = iota
	Backlog
	Documentation
	Done
	InWork
	People
	Review
	Server
)

func (i Icon_e) String() string {
	switch i {
	case Backlog:
		return "backlog"
	case Documentation:
		return "documentation"
	case Done:
		return "done"
	case InWork:
		return "in-work"
	case People:
		return "people"
	case Review:
		return "review"
	case Server:
		return "server"
	default:
		return "none"
	}
}
