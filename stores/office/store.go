// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package office

import (
	"bytes"
	"fmt"
	"regexp"
)

// NewStore returns a new store from the Word document.
func NewStore(path string, reader *bytes.Reader, invalidCharacters, preprocess, sensitiveData, smartQuotes bool) (d *DOCX, err error) {
	dx, err := openDocxReader(reader)
	if err != nil {
		return nil, err
	}

	// convert the xml data to a slice of word tokens
	dx.GenWordsList()

	// convert the word tokens to a slice containing all the words.
	// we collapse spaces into a single space and can't tell the difference between
	// a space and a tab. we also destroy all the original Word tables.
	result := &bytes.Buffer{}
	for _, word := range dx.WordsList {
		//result.WriteString(fmt.Sprintf("%06d: ", line+1))
		for column, content := range word.Content {
			if column != 0 {
				result.WriteString(" ")
			}
			result.WriteString(content)
		}
		result.WriteByte('\n')
	}

	// split the result into lines terminated by newlines
	d = &DOCX{
		path:  path,
		lines: bytes.Split(result.Bytes(), []byte{'\n'}),
	}

	if sensitiveData {
		d.RemoveSensitiveData()
	}
	if preprocess {
		d.Preprocess()
	}

	return d, nil
}

// Lines returns the lines of the document.
// Note that Word tables are not preserved and most runs of spaces and tabs are
// collapsed into a single space.
func (d *DOCX) Lines() [][]byte {
	return d.lines
}

func (d *DOCX) Preprocess() {
	for n, line := range d.lines {
		d.lines[n] = preprocessLine(line)
	}
}

// RemoveSensitiveData removes sensitive data from the document.
// This is destructive and updates the internal lines
func (d *DOCX) RemoveSensitiveData() {
	var lines [][]byte
	var statusLinePrefix, unitId []byte
	for _, line := range d.Lines() {
		if rxCourierSection.Match(line) {
			if len(lines) != 0 {
				lines = append(lines, []byte{})
			}
			lines = append(lines, line)
			unitId = line[8:14]
			statusLinePrefix = []byte(fmt.Sprintf("%s Status: ", unitId))
		} else if rxElementSection.Match(line) {
			if len(lines) != 0 {
				lines = append(lines, []byte{})
			}
			lines = append(lines, line)
			unitId = line[8:14]
			statusLinePrefix = []byte(fmt.Sprintf("%s Status: ", unitId))
		} else if rxFleetSection.Match(line) {
			if len(lines) != 0 {
				lines = append(lines, []byte{})
			}
			lines = append(lines, line)
			unitId = line[6:12]
			statusLinePrefix = []byte(fmt.Sprintf("%s Status: ", unitId))
		} else if rxGarrisonSection.Match(line) {
			if len(lines) != 0 {
				lines = append(lines, []byte{})
			}
			lines = append(lines, line)
			unitId = line[9:15]
			statusLinePrefix = []byte(fmt.Sprintf("%s Status: ", unitId))
		} else if rxTribeSection.Match(line) {
			if len(lines) != 0 {
				lines = append(lines, []byte{})
			}
			lines = append(lines, line)
			unitId = line[6:10]
			statusLinePrefix = []byte(fmt.Sprintf("%s Status: ", unitId))
		} else if bytes.HasPrefix(line, []byte("Current Turn ")) {
			lines = append(lines, line)
		} else if rxFleetMovement.Match(line) {
			lines = append(lines, line)
		} else if bytes.HasPrefix(line, []byte("Tribe Follows ")) {
			lines = append(lines, line)
		} else if bytes.HasPrefix(line, []byte("Tribe Goes to ")) {
			lines = append(lines, line)
		} else if bytes.HasPrefix(line, []byte("Tribe Movement: ")) {
			lines = append(lines, line)
		} else if rxScoutLine.Match(line) {
			lines = append(lines, line)
		} else if statusLinePrefix != nil && bytes.HasPrefix(line, statusLinePrefix) {
			lines = append(lines, line)
		}
	}
	d.lines = lines
}

// todo: can we use this office package if we fix the tab issue?
type DOCX struct {
	path  string
	lines [][]byte
}

var (
	rxCourierSection  = regexp.MustCompile(`^Courier \d{4}c\d *, `)
	rxElementSection  = regexp.MustCompile(`^Element \d{4}e\d *, `)
	rxFleetSection    = regexp.MustCompile(`^Fleet \d{4}f\d *, `)
	rxFleetMovement   = regexp.MustCompile(`^(CALM|MILD|STRONG|GALE) (NE|SE|SW|NW|N|S) Fleet Movement: Move `)
	rxGarrisonSection = regexp.MustCompile(`^Garrison \d{4}g\d *, `)
	rxScoutLine       = regexp.MustCompile(`^Scout \d:Scout `)
	rxTribeSection    = regexp.MustCompile(`^Tribe \d{4} *, `)

	rxUnitHeader = regexp.MustCompile(`^(?:Courier|Element|Fleet|Garrison|Tribe) \d{4}(?:[cefg]\d)?,`)

	reBackslashDash = regexp.MustCompile(`\\+ *-`)

	reSpaces         = regexp.MustCompile(` +`)
	reSpacesLeading  = regexp.MustCompile(` ([,()\\:])`)
	reSpacesTrailing = regexp.MustCompile(`([,()\\:]) `)

	reBackslashUnit = regexp.MustCompile(`\\+(\d{4}(?:[cefg]\d)?)`)
	reDirectionUnit = regexp.MustCompile(`(NE|SE|SW|NW|N|S) (\d{4}(?:[cefg]\d)?)`)

	reRunOfBackslashes = regexp.MustCompile(`\\\\+`)
	reRunOfComma       = regexp.MustCompile(`,,+`)
)

func preprocessLine(line []byte) []byte {
	// remove backslash-dashes
	line = reBackslashDash.ReplaceAll(line, []byte{'\\'})

	// reduce consecutive spaces to a single space
	line = reSpaces.ReplaceAll(line, []byte{' '})

	// remove leading and trailing spaces around some punctuation
	line = reSpacesLeading.ReplaceAll(line, []byte{'$', '1'})
	line = reSpacesTrailing.ReplaceAll(line, []byte{'$', '1'})

	// fix issues with backslash or direction followed by a unit ID
	line = reBackslashUnit.ReplaceAll(line, []byte{',', '$', '1'})
	line = reDirectionUnit.ReplaceAll(line, []byte{'$', '1', ',', '$', '2'})

	// reduce runs of certain punctuation to a single punctuation character
	line = reRunOfBackslashes.ReplaceAll(line, []byte{'\\'})
	line = reRunOfComma.ReplaceAll(line, []byte{','})

	// tweak the fleet movement to remove the trailing comma from the observations
	line = bytes.ReplaceAll(line, []byte{',', ')'}, []byte{')'})

	// remove all trailing backslashes from the line
	line = bytes.TrimRight(line, "\\")

	// the code to reduce runs of commas removes the status field from the unit header.
	// we need to re-add it or the parser will fail to find the unit header.
	if rxUnitHeader.Match(line) {
		header := bytes.Split(line, []byte{','})
		if len(header) == 3 && bytes.HasPrefix(header[1], []byte("Current Hex =")) {
			header = [][]byte{header[0], []byte{}, header[1], header[2]}
			line = bytes.Join(header, []byte{','})
		}
	}
	// and the parser expects a space before the turn number
	if bytes.HasPrefix(line, []byte("Current Turn ")) {
		line = bytes.ReplaceAll(line, []byte{'(', '#'}, []byte{' ', '(', '#'})
	}

	return line
}
