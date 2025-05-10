// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mdhender/ottoapp/components/app"
	"github.com/mdhender/ottoapp/components/app/pages/reports/uploads/docx"
	"github.com/mdhender/ottoapp/components/app/widgets"
	"github.com/playbymail/tndocx"
	dokx "github.com/playbymail/tndocx/docx"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) getReportsDocxUpload(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "reports", "uploads", "docx", "content.gohtml"),
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
			Content: docx.Content_t{},
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

func (s *Server) postDocxUpload(path string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
	}

	render := func(w http.ResponseWriter, r *http.Request, title, message string, button widgets.Button_e) (int, error) {
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
		return s.writeFragments(w, r, alertFragment)
	}
	_ = render

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)

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
			if _, err := render(w, r, "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is missing.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !sb.IsDir() {
			if _, err := render(w, r, "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is not a folder.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		// parse the form data, limiting the size to 1MB
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			if _, err := render(w, r, "Upload failed", "The file upload failed. Please try again with a smaller file.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		// verify that we have exactly one file in the form data. retrieve if we do, otherwise return an error.
		const fieldName = "docx-upload"
		if n := len(r.MultipartForm.File[fieldName]); n == 0 {
			if _, err := render(w, r, "Upload failed", "The file upload failed. We could not find the file in the request. Please try again with a file.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if n > 1 { // it is an error to upload multiple files
			if _, err := render(w, r, "Upload failed", "The file upload failed because the request contained multiple files. Please try again with a single file.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		file, handler, err := r.FormFile(fieldName)
		if err != nil {
			log.Printf("%s %s: parsing form: %v\n", r.Method, r.URL.Path, err)
			if _, err := render(w, r, "Upload failed", "The file upload failed. We were unable to extract the file from the upload request. Please report this error.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		defer func() {
			_ = file.Close()
		}()
		// ensure the uploaded file has the correct suffix
		if !strings.HasSuffix(handler.Filename, ".docx") {
			if _, err := render(w, r, "Invalid file name", "The report file name must end with .docx.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		// load the file into memory
		started := time.Now()
		data, err := io.ReadAll(file)
		if err != nil {
			log.Printf("%s %s: reading form data: %v\n", r.Method, r.URL.Path, err)
			if _, err := render(w, r, "Server error", "The server encountered an error while reading the form data from your request.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if len(data) == 0 {
			if _, err := render(w, r, "Report is empty", "The file uploaded is empty. Please select a different file.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		log.Printf("%s %s: read     %d bytes\n", r.Method, r.URL.Path, len(data))

		// load the Word document
		log.Printf("%s %s: loaded %d bytes in %v\n", r.Method, r.URL.Path, len(data), time.Since(started))

		// extract the text from the Word document
		text, err := dokx.ReadBuffer(data)
		if err != nil {
			log.Printf("%s %s: docx reading buffer: %v\n", r.Method, r.URL.Path, err)
			if _, err := render(w, r, "Server error", "The server encountered an error while reading the Word document uploaded with your request. Please report this error.", ""); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		log.Printf("%s %s: read    %d in %v\n", r.Method, r.URL.Path, len(text), time.Since(started))

		// compress spaces within the text
		text = tndocx.CompressSpaces(text)
		log.Printf("%s %s: despaced to %d bytes in %v\n", r.Method, r.URL.Path, len(text), time.Since(started))

		// remove unnecessary lines from the text
		lines := bytes.Split(text, []byte{'\n'})
		log.Printf("%s %s: split into %d lines in %v\n", r.Method, r.URL.Path, len(lines), time.Since(started))
		lines = tndocx.RemoveNonMappingLines(lines)
		log.Printf("%s %s: trimmed to %d lines in %v\n", r.Method, r.URL.Path, len(lines), time.Since(started))
		for i := range lines {
			lines[i] = tndocx.PreProcessMovementLine(lines[i])
		}
		log.Printf("%s %s: prepped %d lines in %v\n", r.Method, r.URL.Path, len(lines), time.Since(started))

		//// convert the text to a report
		//report := tndocx.ToReport("yyyy-mm", lines)
		//log.Printf("%s %s: created report with %d units in %v\n", r.Method, r.URL.Path, len(report.Units), time.Since(started))
		//
		//// create the json
		//jsonPath := filepath.Join(inputPath, "docx-to-text.json")
		//if buf, err := json.MarshalIndent(report, "", "  "); err != nil {
		//	log.Printf("%s %s: docx marshalling json: %v\n", r.Method, r.URL.Path, err)
		//	if _, err := render(w, r, "Server error", "The server encountered an error while translating your report data to an internal format. Please report this error.", ""); err != nil {
		//		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		//	}
		//	return
		//} else if err := os.WriteFile(jsonPath, buf, 0644); err != nil {
		//	log.Printf("%s %s: docx writing json: %v\n", r.Method, r.URL.Path, err)
		//	if _, err := render(w, r, "Server error", "The server encountered an error while saving your report to disc. Please report this error.", ""); err != nil {
		//		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		//	}
		//}
		//log.Printf("%s: %s: created %s in %v\n", r.Method, r.URL.Path, jsonPath, time.Since(started))
		//
		//if _, err := render(w, r, "Document uploaded and converted to json", "The entire process is not yet fully implemented, but you can view the work in progress.", widgets.BBetaPeekAtJson); err != nil {
		//	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		//}

		_, _ = render(w, r, "Under Construction", "This page is under construction. Some parts of it are not yet implemented.", "")
	}
}
