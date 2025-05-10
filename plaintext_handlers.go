// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mdhender/ottoweb/components/app"
	"github.com/mdhender/ottoweb/components/app/pages/reports/uploads/plaintext"
	"github.com/mdhender/ottoweb/components/app/widgets"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (s *Server) getReportsUploadsPlainText(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "reports", "uploads", "plaintext", "content.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
		filepath.Join(path, "app", "widgets", "report_text.gohtml"),
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
			Content: plaintext.Content_t{},
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

func (s *Server) postPlainTextUpload(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
		filepath.Join(path, "app", "widgets", "report_text.gohtml"),
	}
	const fieldName = "text"

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
		//log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		started, bytesWritten := time.Now(), 0
		//defer func() {
		//	log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		//}()
		_, _ = started, bytesWritten

		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if contentType := r.Header.Get("Content-Type"); !(contentType == "application/x-www-form-urlencoded" || strings.HasPrefix(contentType, "application/x-www-form-urlencoded;")) { // Check the content type
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

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

		// verify that we have an input directory for the clan
		inputPath := filepath.Join(user.Data, "input")
		if sb, err := os.Stat(inputPath); err != nil {
			//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, "", "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is missing.", "")
			if err != nil {
				//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !sb.IsDir() {
			//log.Printf("%s %s: %s is not a directory\n", r.Method, r.URL.Path, inputPath)
			bytesWritten, err = render(w, r, "", "Account error", "Your account has not been set up correctly. Please let the administrator know that your input directory is not a folder.", "")
			if err != nil {
				//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		//log.Printf("%s %s: inputPath %q\n", r.Method, r.URL.Path, inputPath)

		// verify that we have an output directory for the clan
		outputPath := filepath.Join(user.Data, "output")
		if sb, err := os.Stat(outputPath); err != nil {
			//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, "", "Account error", "Your account has not been set up correctly. Please let the administrator know that your output directory is missing.", "")
			if err != nil {
				//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !sb.IsDir() {
			//log.Printf("%s %s: %s is not a directory\n", r.Method, r.URL.Path, outputPath)
			bytesWritten, err = render(w, r, "", "Account error", "Your account has not been set up correctly. Please let the administrator know that your output directory is not a folder.", "")
			if err != nil {
				//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		//log.Printf("%s %s: outputPath %q\n", r.Method, r.URL.Path, outputPath)

		// pull the parameters from the form
		text := r.FormValue(fieldName)
		//log.Printf("%s %s: text %d bytes\n", r.Method, r.URL.Path, len(text))
		if len(text) == 0 {
			//log.Printf("%s %s: text is empty\n", r.Method, r.URL.Path)
			bytesWritten, err = render(w, r, text, "Input is empty", "Please copy your input into the text box and try again.", "")
			if err != nil {
				//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		text = scrubEOL(text)
		//log.Printf("%s %s: text %d bytes\n", r.Method, r.URL.Path, len(text))
		lines := trimLeadingBlankLines(trimTrailingBlankLines(bytes.Split([]byte(text), []byte{'\n'})))
		//log.Printf("%s %s: text %d lines\n", r.Method, r.URL.Path, len(lines))
		if len(lines) < 2 {
			bytesWritten, err = render(w, r, text, "Input is too short", "Expected at least two lines of input.", "")
			if err != nil {
				//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		unitId, turnId, err := checkPlainTextReport(lines)
		if err != nil {
			//log.Printf("%s %s: checkPlainTextReport failed\n", r.Method, r.URL.Path)
			//log.Printf("%s %s: checkPlainTextReport %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, text, "Input checks failed", "Error: "+err.Error(), "")
			if err != nil {
				//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		//log.Printf("%s %s: unitId %q turnId %q\n", r.Method, r.URL.Path, unitId, turnId)

		fileName := fmt.Sprintf("%s.%s.report.txt", turnId, unitId)
		//log.Printf("%s %s: reportFileName %q\n", r.Method, r.URL.Path, fileName)
		reportFile := filepath.Join(inputPath, fileName)
		//log.Printf("%s %s: creating %q\n", r.Method, r.URL.Path, reportFile)

		data := replaceInvalidUTF8(bytes.Join(lines, []byte{'\n'}))
		if len(data) == 0 || data[len(data)-1] != '\n' {
			data = append(data, '\n')
		}
		if err := os.WriteFile(reportFile, data, 0644); err != nil {
			//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, text, "Upload failed", "Error: internal server error!", "")
			if err != nil {
				//log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		//log.Printf("%s %s: created  %q\n", r.Method, r.URL.Path, reportFile)
		//log.Printf("%s %s: wrote    %d bytes\n", r.Method, r.URL.Path, len(data))

		bytesWritten, err = render(w, r, text, "File uploaded", fmt.Sprintf("The uploaded file has been saved as %q. You can view it from the dashboard.", fileName), widgets.BOpenDashboard)
		if err != nil {
			//log.Printf("%s %s: render %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func (s *Server) postPlainTextScrub(path, userdata string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
		filepath.Join(path, "app", "widgets", "report_text.gohtml"),
	}
	fieldName := "text"

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

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		started, bytesWritten := time.Now(), 0
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if contentType := r.Header.Get("Content-Type"); !(contentType == "application/x-www-form-urlencoded" || strings.HasPrefix(contentType, "application/x-www-form-urlencoded;")) { // Check the content type
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

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

		// verify that we have a log directory for the clan
		logsPath := filepath.Join(user.Data, "logs")
		if sb, err := os.Stat(logsPath); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, "", "Account error", "Your account has not been set up correctly. Please let the administrator know that your logs directory is missing.", "")
			if err != nil {
				log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		} else if !sb.IsDir() {
			log.Printf("%s %s: %s is not a directory\n", r.Method, r.URL.Path, logsPath)
			bytesWritten, err = render(w, r, "", "Account error", "Your account has not been set up correctly. Please let the administrator know that your logs directory is not a folder.", "")
			if err != nil {
				log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		log.Printf("%s %s: logsPath %q\n", r.Method, r.URL.Path, logsPath)

		// pull the parameters from the form
		text := r.FormValue(fieldName)
		log.Printf("%s %s: text %d bytes\n", r.Method, r.URL.Path, len(text))
		if len(text) == 0 {
			log.Printf("%s %s: text is empty\n", r.Method, r.URL.Path)
			bytesWritten, err = render(w, r, text, "Input is empty", "Please copy your input into the text box and try again.", "")
			if err != nil {
				log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		rawFile := filepath.Join(logsPath, "_raw_file.txt")
		if err := os.WriteFile(rawFile, []byte(text), 0644); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, text, "File error", "Unable to create a temporary raw file for scrubbing. Please let the administrator know.", "")
			if err != nil {
				log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		log.Printf("%s %s: created %q\n", r.Method, r.URL.Path, rawFile)

		text = scrubEOL(text)
		log.Printf("%s %s: text %d bytes\n", r.Method, r.URL.Path, len(text))

		scrubFile := filepath.Join(logsPath, "_scrub.txt")
		if err := os.WriteFile(scrubFile, []byte(text), 0644); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			bytesWritten, err = render(w, r, text, "File error", "Unable to create a temporary scrubbed file. Please let the administrator know.", "")
			if err != nil {
				log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		log.Printf("%s %s: created %q\n", r.Method, r.URL.Path, scrubFile)

		text += "\n\n***This is a scrubbed file.***\n\n"
		bytesWritten, err = render(w, r, text, "File scrubbed", "Please review the scrubbed file. If it looks correct, you may try to upload it.", "")
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func scrubEOL(input string) string {
	// convert eol on the input file
	var lines [][]byte
	if data := []byte(input); len(data) > 0 {
		data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
		data = bytes.ReplaceAll(data, []byte{'\r'}, []byte{'\n'})
		lines = bytes.Split(data, []byte{'\n'})
		for n, line := range lines {
			lines[n] = bytes.TrimLeft(line, " \t\r\n")
		}
		lines = trimLeadingBlankLines(trimTrailingBlankLines(lines))
	}
	return string(bytes.Join(lines, []byte{'\n'}))
}

func checkUnitHeader(line string) (clanId string, err error) {
	line = strings.ToLower(line)
	//log.Printf("checkUnitHeader: %q\n", line)

	unitId, ok := matchUnitPrefix(line)
	if !ok {
		return "", fmt.Errorf(`first line is missing the unit. expected it to look like "Tribe 0987, ,Current Hex = QQ 1234, (Previous Hex = QQ 1234)"`)
	}

	var currentHex, previousHex string
	switch fields := strings.Split(line, ","); len(fields) {
	case 4:
		currentHex = strings.TrimSpace(fields[2])
		previousHex = strings.TrimSpace(fields[3])
	case 3, 2, 1:
		return "", fmt.Errorf(`first line is missing fields. expected it to look like "Tribe 0987, ,Current Hex = QQ 1234, (Previous Hex = QQ 1234)"`)
	default:
		return "", fmt.Errorf(`first line contains too many fields. expected 4, found %d`, len(fields))
	}

	if !strings.HasPrefix(currentHex, "current hex = ") {
		//log.Printf("checkUnitHeader: currentHex %q\n", currentHex)
		return "", fmt.Errorf(`first line is missing the current hex. expected it to look like "Tribe 0987, ,Current Hex = QQ 1234, (Previous Hex = QQ 1234)"`)
	} else if !strings.HasPrefix(previousHex, "(previous hex = ") {
		//log.Printf("checkUnitHeader: previousHex %q\n", previousHex)
		return "", fmt.Errorf(`first line is missing the previous hex. expected it to look like "Tribe 0987, ,Current Hex = QQ 1234, (Previous Hex = QQ 1234)"`)
	}

	return unitId, nil
}

var (
	rxCourierPrefix  = regexp.MustCompile(`^courier (\d{4}c\d),`)
	rxElementPrefix  = regexp.MustCompile(`^element (\d{4}e\d),`)
	rxFleetPrefix    = regexp.MustCompile(`^fleet (\d{4}f\d),`)
	rxGarrisonPrefix = regexp.MustCompile(`^garrison (\d{4}g\d),`)
	rxTribePrefix    = regexp.MustCompile(`^tribe (\d{4}),`)
)

func matchUnitPrefix(line string) (string, bool) {
	for _, rx := range []*regexp.Regexp{rxTribePrefix, rxCourierPrefix, rxElementPrefix, rxFleetPrefix, rxGarrisonPrefix} {
		if matches := rx.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1], true
		}
	}
	return "", false
}

var (
	rxTurnHeader = regexp.MustCompile(`^current turn (\d+)-(\d+)`)
)

func matchTurnHeader(line string) (string, bool) {
	matches := rxTurnHeader.FindStringSubmatch(line)
	if len(matches) > 2 {
		year, err := strconv.Atoi(matches[1])
		if err != nil {
			return "", false
		} else if year < 899 || year > 1234 {
			return "", false
		}
		month, err := strconv.Atoi(matches[2])
		if err != nil {
			return "", false
		} else if month < 1 || month > 12 {
			return "", false
		}
		return fmt.Sprintf("%04d-%02d", year, month), true
	}
	return "", false
}

func checkTurnHeader(line string) (string, error) {
	line = strings.ToLower(line)
	//log.Printf("checkUnitHeader: %q\n", line)

	turnId, ok := matchTurnHeader(line)
	if !ok {
		return "", fmt.Errorf(`second line is missing Current Turn. expected it to look like "Current Turn 899-12 (#0), Winter, FINE\tNext Turn 900-01 (#1), 29/10/2023"`)
	}

	return turnId, nil
}

// assumes that the input file has been scrubbed and is in plain text
func checkPlainTextReport(lines [][]byte) (unitId, turnId string, err error) {
	if len(lines) < 2 {
		return "", "", fmt.Errorf("input file is missing the tribe and turn lines")
	}

	// extract the clan and turn from the first two lines of the input
	unitId, err = checkUnitHeader(string(lines[0]))
	if err != nil {
		return "", "", err
	}
	turnId, err = checkTurnHeader(string(lines[1]))
	if err != nil {
		return "", "", err
	}

	return unitId, turnId, nil
}
