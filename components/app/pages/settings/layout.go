// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package settings

import "github.com/mdhender/ottoapp/components/app"

type Layout_t struct {
	Title       string
	Tab         string
	ClanId      string
	CurrentPage struct {
		General       bool
		Security      bool
		Plans         bool
		Notifications bool
	}
	Content any
	Footer  app.Footer
}
