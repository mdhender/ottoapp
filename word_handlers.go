// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mdhender/ottoweb/components/app"
	"github.com/mdhender/ottoweb/components/app/pages/reports/uploads"
	"github.com/mdhender/ottoweb/components/app/widgets"
	"github.com/mdhender/ottoweb/stores/office"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func (s *Server) getReportsUploadsMSWord(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "reports", "uploads", "msword", "content.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
	}
	scripts := []string{
		// "https://unpkg.com/jszip/dist/jszip.min.js",
		"/js/jszip-3.10.1.min.js",
		"/js/docx-preview.min.js",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// todo: put auditing info behind a flag
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		started, bytesWritten := time.Now(), 0
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		// fetch the session and get the current user. if either fails, return an error
		user, err := s.extractSession(r)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		payload := app.Layout{
			Title:   fmt.Sprintf("Clan %s", user.Clan),
			Heading: "Reports",
			Scripts: scripts,
			Content: uploads.Content_t{},
			Footer:  footer,
			//Notifications: []widgets.Notification_t{
			//	widgets.Notification_t{
			//		Title:   "Report uploaded",
			//		Message: "You can view the report logs on the Dashboard.",
			//		Button:  widgets.BOpenDashboard,
			//	},
			//},
		}
		payload.CurrentPage.Reports = true
		payload.Footer.Timestamp = time.Now().In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02 15:04:05")

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: parsed components\n", r.Method, r.URL.Path)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.Execute(buf, payload); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		bytesWritten, _ = w.Write(buf.Bytes())
		bytesWritten = len(buf.Bytes())
	}
}

func (s *Server) postReportsUploadsMSWord(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
		filepath.Join(path, "app", "widgets", "report_text.gohtml"),
	}
	fieldName := "file-upload"

	var (
		rxCourierSection  = regexp.MustCompile(`^Courier \d{4}c\d,`)
		rxElementSection  = regexp.MustCompile(`^Element \d{4}e\d,`)
		rxFleetSection    = regexp.MustCompile(`^Fleet \d{4}f\d,`)
		rxFleetMovement   = regexp.MustCompile(`^(CALM|MILD|STRONG|GALE) (NE|SE|SW|NW|N|S) Fleet Movement: Move `)
		rxGarrisonSection = regexp.MustCompile(`^Garrison \d{4}g\d,`)
		rxScoutLine       = regexp.MustCompile(`^Scout \d:Scout `)
		rxTribeSection    = regexp.MustCompile(`^Tribe \d{4},`)
	)

	render := func(w http.ResponseWriter, r *http.Request, text, title, message string, button widgets.Button_e) (int, error) {
		textFragment, err := s.renderFragment(text, "report-text", files...)
		if err != nil {
			return 0, err
		}
		alertFragment, err := s.renderFragment(widgets.NotificationPanel_t{
			OOB: true,
			Notifications: []widgets.Notification_t{{
				Title:   title,
				Message: message,
				Button:  button,
			}},
		}, "notifications-panel", files...)
		if err != nil {
			return 0, err
		}
		return s.writeFragments(w, r, textFragment, alertFragment)
	}
	_ = render

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		log.Printf("%s %s: ct %+v\n", r.Method, r.URL.Path, r.Header.Get("Content-Type"))

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
		log.Printf("%s %s: ct accepted\n", r.Method, r.URL.Path)

		started, bytesWritten := time.Now(), 0
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

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
			bytesWritten, err = render(w, r, "", "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is missing.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !sb.IsDir() {
			log.Printf("%s %s: %s is not a directory\n", r.Method, r.URL.Path, inputPath)
			bytesWritten, err = render(w, r, "", "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is not a folder.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		// pull the parameters from the form
		args := struct {
			invalidCharacters bool
			preprocess        bool
			sensitiveData     bool
			smartQuotes       bool
		}{
			invalidCharacters: cbIsSet(r.FormValue("invalid-characters")),
			preprocess:        true,
			sensitiveData:     true, // cbIsSet(r.FormValue("sensitive-data")),
			smartQuotes:       cbIsSet(r.FormValue("smart-quotes")),
		}

		// parse the form data, limiting the size to 1MB
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			log.Printf("%s %s: parse multi-part form: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, "", "Upload failed", "The file upload failed. Please try again with a smaller file.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		// verify that we have exactly one file in the form data
		if n := len(r.MultipartForm.File[fieldName]); n == 0 {
			bytesWritten, err = render(w, r, "", "Upload failed", "The file upload failed. We could not find the file in the request. Please try again with a file.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if n > 1 { // it is an error to upload multiple files
			bytesWritten, err = render(w, r, "", "Upload failed", "The file upload failed because the request contained multiple files. Please try again with a single file.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		// retrieve the file from the form. the client must send the file in the "report-file" field
		file, handler, err := r.FormFile(fieldName)
		if err != nil {
			log.Printf("%s %s: parsing form: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, "", "Upload failed", "The file upload failed. We were unable to extract the file from the upload request. Please report this error.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		defer func() {
			_ = file.Close()
		}()
		// ensure the uploaded file has the correct suffix
		if !strings.HasSuffix(handler.Filename, ".docx") {
			bytesWritten, err = render(w, r, "", "Invalid file name", "The report file name must end with .docx.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		// load the file into memory
		data, err := io.ReadAll(file)
		if err != nil {
			log.Printf("%s %s: reading form data: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, "", "Server error", "The server encountered an error while reading the form data from your request.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if len(data) == 0 {
			bytesWritten, err = render(w, r, "", "Report is empty", "The file uploaded is empty. Please select a different file.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		log.Printf("%s %s: read     %d bytes\n", r.Method, r.URL.Path, len(data))

		var lines [][]byte
		dss, err := office.NewStore(handler.Filename, bytes.NewReader(data), args.invalidCharacters, args.preprocess, args.sensitiveData, args.smartQuotes)
		if err != nil {
			log.Printf("%s %s: office: reading: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, "", "Server error", "The server encountered an error creating the office store. Please report this error.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		if rawLines := dss.Lines(); len(rawLines) == 0 {
			bytesWritten, err = render(w, r, "", "Server error", "The server encountered an error reading empty lines. Please report this error.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else {
			var statusLinePrefix, unitId []byte
			for _, line := range rawLines {
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
		}
		log.Printf("%s %s: daFile: lines %d\n", r.Method, r.URL.Path, len(lines))
		//for n, line := range lines {
		//	log.Printf("%s %s: office: line %d: %q\n", r.Method, r.URL.Path, n, line)
		//	if n > 23 {
		//		break
		//	}
		//}

		// extract the clan and turn from the first two lines of the input
		var clanId, turnId string
		_, _ = clanId, turnId
		if len(lines) < 2 {
			bytesWritten, err = render(w, r, "", "File error", "The report file contained less than the expected two lines.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		//log.Printf("%s %s: clan line %q\n", r.Method, r.URL.Path, string(lines[0]))
		clanFields := bytes.Split(lines[0], []byte{','})
		//log.Printf("%s %s: clan %d\n", r.Method, r.URL.Path, len(clanFields))
		//for n, fld := range clanFields {
		//	log.Printf("%s %s: clan field %d: %q\n", r.Method, r.URL.Path, n, string(fld))
		//}
		if len(clanFields) != 4 {
			bytesWritten, err = render(w, r, "", "File error", "The clan header did not contain exactly four fields.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		tribeField, currHexField, prevHexField := clanFields[0], clanFields[2], clanFields[3]
		if !bytes.HasPrefix(tribeField, []byte("Tribe 0")) {
			//log.Printf("%s %s: clan %+v\n", r.Method, r.URL.Path, clanFields)
			bytesWritten, err = render(w, r, "", "File error", "The clan header has an invalid tribe field. We expect it to start with \"Tribe 0\".", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !bytes.HasPrefix(currHexField, []byte("Current Hex = ")) {
			bytesWritten, err = render(w, r, "", "File error", "The clan header has an invalid Current Hex field. We expect it to contain a location like \"## 0101\" or \"KK 0101\".", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !bytes.HasPrefix(prevHexField, []byte("(Previous Hex = ")) {
			bytesWritten, err = render(w, r, "", "File error", "The clan header has an invalid Previous Hex field. We expect it to contain a location like \"## 0101\", \"KK 0101\" or \"N/A\".", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		if fields := strings.Fields(string(tribeField)); len(fields) != 2 {
			bytesWritten, err = render(w, r, "", "File error", "The Tribe field in the clan header seems to be missing the tribe.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else {
			clanId = fields[1]
			//log.Printf("%s %s: clan fields: clanId %q\n", r.Method, r.URL.Path, clanId)
		}
		//log.Printf("%s %s: turn %q\n", r.Method, r.URL.Path, string(lines[1]))
		turnFields := bytes.Split(lines[1], []byte{','})
		//log.Printf("%s %s: turn %d\n", r.Method, r.URL.Path, len(turnFields))
		//for n, fld := range turnFields {
		//	log.Printf("%s %s: turn field %d: %q\n", r.Method, r.URL.Path, n, string(fld))
		//}
		if len(turnFields) != 4 {
			bytesWritten, err = render(w, r, "", "File error", "The turn data on the second line doesn't contain two fields.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !bytes.HasPrefix(turnFields[0], []byte("Current Turn ")) {
			bytesWritten, err = render(w, r, "", "File error", "The second line of the file did not start with \"Current Turn\".", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		currentTurnFields := strings.Fields(string(turnFields[0]))
		//log.Printf("%s %s: turn fields: currentTurnFields %+v\n", r.Method, r.URL.Path, currentTurnFields)
		if len(currentTurnFields) != 4 {
			bytesWritten, err = render(w, r, "", "File error", "The second line of the file did not contain four fields.", "")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else {
			turnId = currentTurnFields[2]
			//log.Printf("%s %s: turn fields: turnId %q\n", r.Method, r.URL.Path, turnId)
		}

		data = bytes.Join(lines, []byte{'\n'})
		if len(data) == 0 || data[len(data)-1] != '\n' {
			data = append(data, '\n')
		}

		reportFile := filepath.Join(inputPath, "docx-to-text.txt")
		//log.Printf("%s %s: creating %q\n", r.Method, r.URL.Path, reportFile)

		if err := os.WriteFile(reportFile, data, 0644); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		//log.Printf("%s %s: created  %q\n", r.Method, r.URL.Path, reportFile)

		//log.Printf("%s %s: wrote    %d bytes\n", r.Method, r.URL.Path, len(data))

		bytesWritten, err = render(w, r, "", "Document uploaded and converted to text", "The entire process is not yet fully implemented, but you can view the work in progress.", widgets.BBetaPeekAtDocx)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

//func (s *Server) postReportsUploadsMSWordURLEncoded(path string, footer app.Footer) http.HandlerFunc {
//	files := []string{
//		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
//	}
//
//	return func(w http.ResponseWriter, r *http.Request) {
//		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
//		log.Printf("%s %s: ct %+v\n", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
//
//		if r.Method != "POST" {
//			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
//			return
//		}
//
//		if r.Method != "POST" {
//			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
//			return
//		} else if r.Header.Get("HX-Request") != "true" {
//			log.Printf("%s %s: hx-request missing\n", r.Method, r.URL.Path)
//			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
//			return
//		} else if contentType := r.Header.Get("Content-Type"); !(contentType == "application/x-www-form-urlencoded" || strings.HasPrefix(contentType, "application/x-www-form-urlencoded;")) {
//			log.Printf("%s %s: ct %q\n", r.Method, r.URL.Path, contentType)
//			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
//			return
//		}
//
//		started, bytesWritten := time.Now(), 0
//		defer func() {
//			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
//		}()
//
//		contentType := r.Header.Get("Content-Type")
//		log.Printf("%s %s: ct %q\n", r.Method, r.URL.Path, contentType)
//		if !(contentType == "application/x-www-form-urlencoded" || strings.HasPrefix(contentType, "application/x-www-form-urlencoded;")) { // Check the content type
//			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
//			return
//		}
//		log.Printf("%s %s: ct accepted\n", r.Method, r.URL.Path)
//
//		// fetch the session and get the current user. if either fails, return an error
//		user, err := s.extractSession(r)
//		if err != nil && !errors.Is(err, sql.ErrNoRows) {
//			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
//			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
//			return
//		} else if user == nil {
//			// there is no active session, so this is an error
//			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
//			return
//		}
//		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)
//
//		// pull the parameters from the form
//		args := struct {
//			docxInput         string
//			invalidCharacters bool
//			sensitiveData     bool
//			smartQuotes       bool
//		}{
//			docxInput:         r.FormValue("docx-input"),
//			invalidCharacters: cbIsSet(r.FormValue("invalid-characters")),
//			sensitiveData:     cbIsSet(r.FormValue("sensitive-data")),
//			smartQuotes:       cbIsSet(r.FormValue("smart-quotes")),
//		}
//
//		var payload []widgets.Notification_t
//		log.Printf("%s %s: len(docx.input) is %d\n", r.Method, r.URL.Path, len(args.docxInput))
//		if len(args.docxInput) == 0 {
//			payload = append(payload, widgets.Notification_t{
//				Title:   "Report is empty",
//				Message: "The file uploaded is empty. Please select a different file.",
//			})
//		} else {
//			payload = append(payload, widgets.Notification_t{
//				Title:   "Report uploaded",
//				Message: fmt.Sprintf("Your file contained %d bytes. You can view the report logs on the Dashboard.", len(args.docxInput)),
//				Button:  widgets.BOpenDashboard,
//			})
//		}
//
//		t, err := template.ParseFiles(files...)
//		if err != nil {
//			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
//			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
//			return
//		}
//		log.Printf("%s %s: parsed components\n", r.Method, r.URL.Path)
//
//		// parse into a buffer so that we can handle errors without writing to the response
//		buf := &bytes.Buffer{}
//		if err := t.ExecuteTemplate(buf, "notifications-panel", payload); err != nil {
//			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
//			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
//			return
//		}
//
//		w.Header().Set("Content-Type", "text/html; charset=utf-8")
//		w.WriteHeader(http.StatusOK)
//		bytesWritten, _ = w.Write(buf.Bytes())
//		bytesWritten = len(buf.Bytes())
//
//		//w.Header().Set("Content-Type", "text/html; charset=utf-8")
//		//w.WriteHeader(http.StatusOK)
//		//_, _ = w.Write([]byte(`<p>yay</p>`))
//		//return
//		//
//		//http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
//		//return
//	}
//}

func (s *Server) postApiReportUploadDocx(userdata string) http.HandlerFunc {
	const fieldName = "file-upload"

	return func(w http.ResponseWriter, r *http.Request) {
		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		if r.Method != "POST" {
			log.Printf("%s %s: %q != POST\n", r.Method, r.URL.Path, r.Method)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		log.Printf("%s %s: %q == POST\n", r.Method, r.URL.Path, r.Method)

		contentType := r.Header.Get("Content-Type")
		log.Printf("%s %s: ct %q\n", r.Method, r.URL.Path, contentType)
		if !(contentType == "multipart/form-data" || strings.HasPrefix(contentType, "multipart/form-data;")) { // Check the content type
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		log.Printf("%s %s: ct accepted\n", r.Method, r.URL.Path)

		// fetch the session and get the current user. if either fails, return an error
		user, err := s.extractSession(r)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		// verify that we have an input directory for the clan
		inputPath := filepath.Join(user.Data, "input")
		if sb, err := os.Stat(inputPath); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if !sb.IsDir() {
			log.Printf("%s %s: %s is not a directory\n", r.Method, r.URL.Path, inputPath)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: inputPath %q\n", r.Method, r.URL.Path, inputPath)

		// check for the remove-bad-bytes and remove-sensitive-lines parameter in the form data
		removeBadBytes := cbIsSet(r.FormValue("remove-bad-bytes"))
		log.Printf("%s %s: removeBadBytes %v\n", r.Method, r.URL.Path, removeBadBytes)
		sensitiveData := cbIsSet(r.FormValue("sensitive-data"))
		log.Printf("%s %s: sensitiveLines %v\n", r.Method, r.URL.Path, sensitiveData)

		// parse the form data, limiting the size to 1MB
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			log.Printf("%s %s: parse multi-part form: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// verify that we have exactly one file in the form data
		log.Printf("%s %s: files: %d\n", r.Method, r.URL.Path, len(r.MultipartForm.File))
		for k, v := range r.MultipartForm.File {
			log.Printf("%s %s: files: %q: %v\n", r.Method, r.URL.Path, k, len(v))
		}
		if n := len(r.MultipartForm.File[fieldName]); n == 0 {
			log.Printf("%s %s: files: missing %q\n", r.Method, r.URL.Path, fieldName)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if n > 1 { // it is an error to upload multiple files
			log.Printf("%s %s: files: %q != %d\n", r.Method, r.URL.Path, fieldName, n)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		log.Printf("%s %s: files: %q found\n", r.Method, r.URL.Path, fieldName)

		// retrieve the file from the form. the client must send the file in the "report-file" field
		file, handler, err := r.FormFile(fieldName)
		if err != nil {
			log.Printf("%s %s: parsing form: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer func() {
			_ = file.Close()
		}()
		data, err := io.ReadAll(file)
		if err != nil {
			log.Printf("%s %s: reading form data: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: read     %d bytes\n", r.Method, r.URL.Path, len(data))

		// ensure the uploaded file has the correct suffix
		log.Printf("%s %s: filename %q\n", r.Method, r.URL.Path, handler.Filename)

		// send a json response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(struct {
			Success bool `json:"success"`
		}{
			Success: true,
		})
	}
}
