// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package dashboard

import "github.com/mdhender/ottoapp/components/app"

type Content struct {
	ClanId string
	Turns  []*TurnFiles_t
}

// TurnFiles_t represents a turn and the files associated with it.
type TurnFiles_t struct {
	Turn    string            // year-month
	ClanId  string            // clan id contained in the files for this turn
	IsEmpty bool              // true if there are no files for this turn
	Reports []*app.FileInfo_t // empty if no reports
	Errors  []*app.FileInfo_t // empty if no errors
	Logs    []*app.FileInfo_t // empty if no logs
	Maps    []*app.FileInfo_t // empty if no maps
}

// Less returns true if the turn should display before the other turn.
// Sorted by turn (newer before older), then clan id.
func (tf *TurnFiles_t) Less(t *TurnFiles_t) bool {
	if t == nil {
		return true
	}
	// compare turns first
	if tf.Turn > t.Turn {
		return true
	} else if tf.Turn == t.Turn {
		return tf.ClanId < t.ClanId
	}
	return false
}
