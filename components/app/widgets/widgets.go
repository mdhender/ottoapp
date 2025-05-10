// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package widgets

type NotificationPanel_t struct {
	OOB           bool
	Notifications []Notification_t
}

type Notification_t struct {
	Title   string
	Message string
	Button  Button_e
}

type Button_e string

const (
	BNone           Button_e = ""
	BBetaPeekAtDocx Button_e = "beta-peek-at-docx"
	BBetaPeekAtJson Button_e = "beta-peek-at-json"
	BOpenDashboard  Button_e = "open-dashboard"
)

type ReportText_t struct {
	Text string
}
