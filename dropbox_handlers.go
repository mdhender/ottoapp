// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/mdhender/ottoweb/components/app"
	"github.com/mdhender/ottoweb/components/app/pages/reports/uploads/dropbox"
	"github.com/mdhender/ottoweb/components/app/widgets"
	"github.com/playbymail/tndocx"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (s *Server) getReportsDropboxUpload(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "reports", "uploads", "dropbox", "content.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
		filepath.Join(path, "app", "widgets", "report_text.gohtml"),
	}
	scripts := []string{
		"/js/jszip-3.10.1.min.js",
		"/js/docx-preview.min.js",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// todo: put auditing info behind a flag
		//log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		started, bytesWritten := time.Now(), 0
		//defer func() {
		//	log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		//}()
		_, _ = started, bytesWritten

		// fetch the session and get the current user. if either fails, return an error
		user, err := s.extractSession(r)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			//log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		//log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		payload := app.Layout{
			Title:   fmt.Sprintf("Clan %s", user.Clan),
			Heading: "Reports",
			Scripts: scripts,
			Content: dropbox.Content_t{},
			Footer:  footer,
		}
		payload.CurrentPage.Reports = true
		payload.Footer.Timestamp = time.Now().In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02 15:04:05")

		t, err := template.ParseFiles(files...)
		if err != nil {
			//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		//log.Printf("%s %s: parsed components\n", r.Method, r.URL.Path)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.Execute(buf, payload); err != nil {
			//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		bytesWritten, _ = w.Write(buf.Bytes())
		bytesWritten = len(buf.Bytes())
	}
}

func (s *Server) postDropboxScrub(path string, serverVersion string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "pages", "reports", "uploads", "dropbox", "content.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
	}
	alert := func(w http.ResponseWriter, r *http.Request, title, message string, button widgets.Button_e) {
		alertFragment, err := s.renderFragment(widgets.NotificationPanel_t{
			OOB: true,
			Notifications: []widgets.Notification_t{{
				Title:   title,
				Message: message,
				Button:  button,
			}},
		}, "notifications-panel", files...)
		if err != nil {
			return
		}
		_, _ = s.writeFragments(w, r, alertFragment)
		return
	}

	const fieldName = "report-file-input"
	rxTurnReports := regexp.MustCompile(`^([0-9]+)-([0-9]+)\.([0-9]+)\.report\.(docx|txt)$`)

	return func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("%s %s: entered\n", r.Method, r.URL.Path)

		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			log.Printf("%s %s: hx-request missing\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if contentType := r.Header.Get("Content-Type"); !(contentType == "multipart/form-data" || strings.HasPrefix(contentType, "multipart/form-data;")) {
			log.Printf("%s %s: ct %q\n", r.Method, r.URL.Path, contentType)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		// fetch the session and get the current user. if either fails, return an error
		user, err := s.extractSession(r)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// after those header checks, we're expected to return an alert fragment if there are issues with the request

		// verify that we have an input directory for the clan
		inputPath := filepath.Join(user.Data, "input")
		if sb, err := os.Stat(inputPath); err != nil {
			alert(w, r, "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is missing.", "")
			return
		} else if !sb.IsDir() {
			alert(w, r, "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is not a folder.", "")
			return
		}

		// parse the form data, limiting the size to 1MB, and verify that we have exactly one file in the form data.
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			alert(w, r, "Upload failed", "The file upload failed. The attached file exceeds the size limit of 1mb.", "")
			return
		} else if n := len(r.MultipartForm.File[fieldName]); n == 0 {
			alert(w, r, "Upload failed", "The file upload failed. The request did not include a named file.", "")
			return
		} else if n > 1 { // it is an error to upload multiple files
			alert(w, r, "Upload failed", "The file upload failed. The request included multiple files.", "")
			return
		}
		// read the file from the form data
		file, handler, err := r.FormFile(fieldName)
		if err != nil {
			id := uuid.NewString()
			log.Printf("%s %s: parsing form: %v (%s)\n", r.Method, r.URL.Path, err, id)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			alert(w, r, "Upload failed", fmt.Sprintf("The file upload failed. The attached file could not be extracted from the request. Please report error %q", id[len(id)-8:]), "")
			return
		}
		defer func() {
			_ = file.Close()
		}()

		// the file name must match the YYYY-MM.CLAN.report.ext pattern, but I got talked into
		// allowing for three digit years, so we must normalize both the year and month.
		var fileName, reportId, turnId, clanId string
		isTextFile, isWordFile := false, false
		if matches := rxTurnReports.FindStringSubmatch(handler.Filename); len(matches) != 5 {
			alert(w, r, "Upload failed", "The file upload failed. The file name must match YEAR-MONTH.CLAN.report and have an extension of .txt or .docx.", "")
			return
		} else {
			var year, month, clanNo int
			if year, err = strconv.Atoi(matches[1]); err != nil {
				alert(w, r, "Upload failed", "The file upload failed. The file name must include a numeric YEAR.", "")
				return
			} else if year < 899 || year > 1234 {
				alert(w, r, "Upload failed", "The file upload failed. The YEAR in the file must be between 899 and 1234.", "")
				return
			} else if month, err = strconv.Atoi(matches[2]); err != nil {
				alert(w, r, "Upload failed", "The file upload failed. The file name must include a numeric MONTH.", "")
				return
			} else if month < 1 || month > 12 {
				alert(w, r, "Upload failed", "The file upload failed. The MONTH in the file name must be between 1 and 12.", "")
				return
			} else if clanNo, err = strconv.Atoi(matches[3]); err != nil {
				alert(w, r, "Upload failed", "The file upload failed. The file name must include a numeric CLAN.", "")
				return
			} else if clanNo < 1 || clanNo > 999 {
				alert(w, r, "Upload failed", "The file upload failed. The CLAN in the file name must be between 1 and 999.", "")
				return
			}
			ext := matches[4]
			isTextFile, isWordFile = ext == "txt", ext == "docx"
			turnId = fmt.Sprintf("%04d-%02d", year, month)
			clanId = fmt.Sprintf("%04d", clanNo)
			reportId = fmt.Sprintf("%s.%s", turnId, clanId)
			fileName = fmt.Sprintf("%s.report.%s", reportId, ext)
		}
		//log.Printf("%s %s: filename %q\n", r.Method, r.URL.Path, fileName)

		// ensure the uploaded file has the correct content-type based on the extension
		//log.Printf("%s %s: field %q: %q\n", r.Method, r.URL.Path, fieldName, handler.Filename)
		//log.Printf("%s %s: field %q: %v\n", r.Method, r.URL.Path, fieldName, handler.Header["Content-Type"])
		if !(isTextFile || isWordFile) {
			alert(w, r, "Upload failed", "The file upload failed. The extension must be .txt or .docx.", "")
			return
		} else if isTextFile && handler.Header["Content-Type"][0] != "text/plain" {
			//log.Printf("%s %s: field %q: %q: unexpected content type %q\n", r.Method, r.URL.Path, fieldName, handler.Filename, handler.Header["Content-Type"][0])
			alert(w, r, "Upload failed", "The file upload failed. The browser did not encode the text file correctly.", "")
			return
		} else if isWordFile && handler.Header["Content-Type"][0] != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
			//log.Printf("%s %s: field %q: %q: unexpected content type %q\n", r.Method, r.URL.Path, fieldName, handler.Filename, handler.Header["Content-Type"][0])
			alert(w, r, "Upload failed", "The file upload failed. The browser did not encode the word document correctly.", "")
			return
		}

		// load the file into memory
		data, err := io.ReadAll(file)
		if err != nil {
			//log.Printf("%s %s: reading form data: %v\n", r.Method, r.URL.Path, err)
			alert(w, r, "Upload failed", "The file upload failed. We tried to read the file, but failed. This could be a bug...", "")
			return
		} else if len(data) == 0 {
			alert(w, r, "Upload failed", "The file upload failed. The attached file is empty.", "")
			return
		}

		// parse the report text into sections
		sections, err := tndocx.ParseSections(data)
		if err != nil {
			if errors.Is(err, tndocx.ErrEmptyInput) {
				alert(w, r, "Upload failed", "The file upload failed. We could not find any lines in the file.", "")
			} else if errors.Is(err, tndocx.ErrUnknownFormat) {
				alert(w, r, "Upload failed", "The file upload failed. We could not find any report sections in the report text.", "")
			} else {
				log.Printf("%s %s: parse sections: %v\n", r.Method, r.URL.Path, err)
				alert(w, r, "Upload failed", "The file upload failed. We could not parse the report text.", "")
			}
			return
		}
		//log.Printf("%s %s: parsed report with %d units in %v\n", r.Method, r.URL.Path, len(sections), time.Since(started))

		// create a scrubbed file from the sections
		scrubbedData := &bytes.Buffer{}
		if isTextFile {
			scrubbedData.WriteString(fmt.Sprintf("// text file %q\n", fileName))
		} else if isWordFile {
			scrubbedData.WriteString(fmt.Sprintf("// word file %q\n", fileName))
		}
		metaTimestamp := time.Now().In(user.LanguageAndDates.Timezone.Location)
		scrubbedData.WriteString(fmt.Sprintf("// submitted by user %s at %s\n", user.Clan, metaTimestamp.Format("2006-01-02 15:04:05")))
		scrubbedData.WriteString(fmt.Sprintf("// ottoweb v%s\n", serverVersion))
		scrubbedData.WriteString(fmt.Sprintf("// tndocx  v%s\n", tndocx.Version()))
		// stuff the section back in
		for _, section := range sections {
			scrubbedData.WriteString(fmt.Sprintf("\n// section %d\n", section.Id))
			if len(section.Header) == 0 {
				scrubbedData.WriteString("// missing element header")
			} else {
				scrubbedData.Write(section.Header)
			}
			scrubbedData.WriteByte('\n')
			if len(section.Turn) == 0 {
				scrubbedData.WriteString("// missing turn header")
			} else {
				scrubbedData.Write(section.Turn)
			}
			scrubbedData.WriteByte('\n')
			if len(section.Moves.Movement) != 0 {
				scrubbedData.Write(section.Moves.Movement)
				scrubbedData.WriteByte('\n')
			}
			if len(section.Moves.Follows) != 0 {
				scrubbedData.Write(section.Moves.Fleet)
				scrubbedData.WriteByte('\n')
			}
			if len(section.Moves.GoesTo) != 0 {
				scrubbedData.Write(section.Moves.GoesTo)
				scrubbedData.WriteByte('\n')
			}
			if len(section.Moves.Fleet) != 0 {
				scrubbedData.Write(section.Moves.Fleet)
				scrubbedData.WriteByte('\n')
			}
			for _, scout := range section.Moves.Scouts {
				scrubbedData.Write(scout)
				scrubbedData.WriteByte('\n')
			}
			if len(section.Status) == 0 {
				scrubbedData.WriteString("// missing element status")
			} else {
				scrubbedData.Write(section.Status)
			}
			scrubbedData.WriteByte('\n')
		}

		scrubbedPath := filepath.Join(inputPath, fmt.Sprintf("%s.scrubbed.txt", reportId))
		if err := os.WriteFile(scrubbedPath, scrubbedData.Bytes(), 0644); err != nil {
			id := uuid.NewString()
			log.Printf("%s %s: dropbox: writing scrubbed file: %v (%s)\n", r.Method, r.URL.Path, err, id)
			alert(w, r, "Server error", fmt.Sprintf("The server encountered an error while saving your report. Please report error %q.", id[len(id)-8:]), "")
			return
		}
		//log.Printf("%s: %s: created %s in %v\n", r.Method, r.URL.Path, scrubbedPath, time.Since(started))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Redirect", fmt.Sprintf("/reports/turn/%s/clan/%s", turnId, clanId))
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) postDropboxUpload(path string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
	}

	const fieldName = "report-text"

	alert := func(w http.ResponseWriter, r *http.Request, title, message string, button widgets.Button_e) {
		alertFragment, err := s.renderFragment(widgets.NotificationPanel_t{
			OOB: true,
			Notifications: []widgets.Notification_t{{
				Title:   title,
				Message: message,
				Button:  button,
			}},
		}, "notifications-panel", files...)
		if err != nil {
			return
		}
		_, _ = s.writeFragments(w, r, alertFragment)
		return
	}

	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)

		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			log.Printf("%s %s: hx-request missing\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if contentType := r.Header.Get("Content-Type"); !(contentType == "application/x-www-form-urlencoded" || strings.HasPrefix(contentType, "application/x-www-form-urlencoded;")) {
			log.Printf("%s %s: ct %q\n", r.Method, r.URL.Path, contentType)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		log.Printf("%s %s: ct %q: accepted\n", r.Method, r.URL.Path, r.Header.Get("Content-Type"))

		// fetch the session and get the current user. if either fails, return an error
		user, err := s.extractSession(r)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// verify that we have an input directory for the clan
		inputPath := filepath.Join(user.Data, "input")
		if sb, err := os.Stat(inputPath); err != nil {
			alert(w, r, "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is missing.", "")
			return
		} else if !sb.IsDir() {
			alert(w, r, "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is not a folder.", "")
			return
		}

		// pull the report text from the form
		data := []byte(r.FormValue(fieldName))
		//log.Printf("%s %s: text %d bytes\n", r.Method, r.URL.Path, len(data))
		if len(data) == 0 {
			alert(w, r, "Upload failed", "The file upload failed. The request contained an empty document.", "")
			return
		}

		// parse the report text into sections
		sections, err := tndocx.ParseSections(data)
		if err != nil {
			if errors.Is(err, tndocx.ErrEmptyInput) {
				alert(w, r, "Upload failed", "The file upload failed. We could not find any lines in the file.", "")
			} else if errors.Is(err, tndocx.ErrUnknownFormat) {
				alert(w, r, "Upload failed", "The file upload failed. We could not find any report sections in the report text.", "")
			} else {
				log.Printf("%s %s: parse sections: %v\n", r.Method, r.URL.Path, err)
				alert(w, r, "Upload failed", "The file upload failed. We could not parse the report text.", "")
			}
			return
		}
		log.Printf("%s %s: parsed report with %d units in %v\n", r.Method, r.URL.Path, len(sections), time.Since(started))
		//report := tndocx.ToReport("yyyy-mm", bytes.Split(data, []byte{'\n'}))
		//log.Printf("%s %s: created report with %d units in %v\n", r.Method, r.URL.Path, len(report.Units), time.Since(started))
		//log.Printf("%s %s: report: turn %q\n", r.Method, r.URL.Path, report.TurnId)
		//clanId := "9999"
		//for _, unit := range report.Units {
		//	if unit.Id < clanId {
		//		clanId = unit.Id
		//	}
		//}
		//if clanId == "9999" || len(clanId) < 4 {
		//	alert(w, r, "Upload failed", "The file upload failed. We could not find a clan ID in the report.", "")
		//	return
		//} else if n, err := strconv.Atoi(clanId); err != nil || n < 1 {
		//	alert(w, r, "Upload failed", "The file upload failed. We could not find a valid clan ID in the report.", "")
		//	return
		//}
		//clanId = "0" + clanId[1:]
		//log.Printf("%s %s: report: clan %q\n", r.Method, r.URL.Path, clanId)
		//
		//// create the report file
		//reportPath := filepath.Join(inputPath, fmt.Sprintf("%s.%s.report-scrubbed.txt", report.TurnId, clanId))
		//if err := os.WriteFile(reportPath, data, 0644); err != nil {
		//	log.Printf("%s %s: dropbox writing report: %v\n", r.Method, r.URL.Path, err)
		//	alert(w, r, "Server error", "The server encountered an error while saving your report. Please report this error.", "")
		//	return
		//}
		//log.Printf("%s: %s: created %s in %v\n", r.Method, r.URL.Path, reportPath, time.Since(started))

		alert(w, r, "Under Construction", "This page is under construction. Some parts of it are not yet implemented.", "")
	}
}

func containsNonASCII(b []byte) bool {
	var line []byte
	for _, ch := range b {
		if ch == '\n' {
			line = nil
		}
		line = append(line, ch)
		if ch > 127 {
			log.Printf("%s %s: contains non-ASCII byte %d\n", "r.Method", "r.URL.Path", ch)
			log.Printf("%s %s: contains non-ASCII byte %q\n", "r.Method", "r.URL.Path", line)
			return true
		}
	}
	return false
}
