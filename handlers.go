// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mdhender/ottoapp/components/app"
	"github.com/mdhender/ottoapp/components/app/pages/calendar"
	"github.com/mdhender/ottoapp/components/app/pages/dashboard"
	"github.com/mdhender/ottoapp/components/app/pages/reports"
	"github.com/mdhender/ottoapp/components/app/pages/reports/failed"
	"github.com/mdhender/ottoapp/components/app/pages/reports/uploads"
	"github.com/mdhender/ottoapp/components/app/pages/settings"
	"github.com/mdhender/ottoapp/components/app/pages/settings/general"
	"github.com/mdhender/ottoapp/components/app/pages/settings/plans"
	"github.com/mdhender/ottoapp/components/hero"
	"github.com/mdhender/ottoapp/components/pages"
	"github.com/mdhender/ottoapp/domains"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Server) getApiClanFilesV1(path string) http.HandlerFunc {
	log.Printf("getApiClanFilesV1: path %q\n", path)
	//store, err := ffs.New(path)
	//if err != nil {
	//	log.Printf("error: %v\n", err)
	//	return func(w http.ResponseWriter, r *http.Request) {
	//		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	//	}
	//}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		started, bytesWritten := time.Now(), 0
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		} else if clanId := r.PathValue("clan_id"); clanId != user.Clan {
			// do not let users request other clan's data
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		cf, err := s.stores.ffs.GetClanFiles(user)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		buf, _ := json.MarshalIndent(cf, "", "  ")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		bytesWritten, _ = w.Write(buf)
	}
}

func (s *Server) getApiPathsV1() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		started, bytesWritten := time.Now(), 0
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		response := struct {
			Assets     string `json:"assets"`
			Components string `json:"components"`
			Data       string `json:"data"`
		}{
			Assets:     s.paths.assets,
			Components: s.paths.components,
			Data:       s.paths.userdata,
		}

		buf, _ := json.MarshalIndent(response, "", "  ")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		bytesWritten, _ = w.Write(buf)
	}
}

func (s *Server) postApiReportUploadFile(userdata string) http.HandlerFunc {
	const fieldName = "report-file"
	rxTurnReports := regexp.MustCompile(`^([0-9]+)-([0-9]+)\.([0-9]+)\.report\.txt`)

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

		// check for the remove-bad-bytes and remove-sensitive-lines parameter in the form data
		removeBadBytes := cbIsSet(r.FormValue("remove-bad-bytes"))
		log.Printf("%s %s: removeBadBytes %v\n", r.Method, r.URL.Path, removeBadBytes)
		removeSensitiveLines := cbIsSet(r.FormValue("remove-sensitive-lines"))
		log.Printf("%s %s: removeSensitiveLines %v\n", r.Method, r.URL.Path, removeSensitiveLines)

		// parse the form data, limiting the size to 1MB
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// it is an error to upload multiple files
		if n := len(r.MultipartForm.File[fieldName]); n != 1 {
			log.Printf("%s %s: files %d\n", r.Method, r.URL.Path, n)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// retrieve the file from the form. the client must send the file in the "report-file" field
		file, handler, err := r.FormFile(fieldName)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// ensure the uploaded file has the correct suffix
		log.Printf("%s %s: filename %q\n", r.Method, r.URL.Path, handler.Filename)
		if !strings.HasSuffix(handler.Filename, ".report.txt") {
			log.Printf("%s %s: suffix %q\n", r.Method, r.URL.Path, handler.Filename)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		var fileName string
		if matches := rxTurnReports.FindStringSubmatch(handler.Filename); len(matches) != 4 {
			log.Printf("%s %s: matches %d\n", r.Method, r.URL.Path, len(matches))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else {
			var year, month, clanId int
			if year, err = strconv.Atoi(matches[1]); err != nil {
				log.Printf("%s %s: year %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			} else if year < 899 || year > 1234 {
				log.Printf("%s %s: year %d\n", r.Method, r.URL.Path, year)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			} else if month, err = strconv.Atoi(matches[2]); err != nil {
				log.Printf("%s %s: month %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			} else if month < 1 || month > 12 {
				log.Printf("%s %s: month %d\n", r.Method, r.URL.Path, month)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			} else if clanId, err = strconv.Atoi(matches[3]); err != nil {
				log.Printf("%s %s: clan %v\n", r.Method, r.URL.Path, err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			} else if clanId < 1 || clanId > 999 {
				log.Printf("%s %s: clan %d\n", r.Method, r.URL.Path, month)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			fileName = fmt.Sprintf("%04d-%02d.%04d.report.txt", year, month, clanId)
		}

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

		// convert eol on the input file
		data, err := io.ReadAll(file)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else {
			data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
			data = bytes.ReplaceAll(data, []byte{'\r'}, []byte{'\n'})
		}

		reportFile := filepath.Join(inputPath, fileName)
		log.Printf("%s %s: creating %q\n", r.Method, r.URL.Path, reportFile)

		if err := os.WriteFile(reportFile, data, 0644); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: created  %q\n", r.Method, r.URL.Path, reportFile)

		bytesWritten = len(data)

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

func (s *Server) postApiReportUploadText(userdata string) http.HandlerFunc {
	const fieldName = "report-file"
	rxTurnReports := regexp.MustCompile(`^([0-9]+)-([0-9]+)\.([0-9]+)\.report\.txt`)

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
		if !(contentType == "application/x-www-form-urlencoded" || strings.HasPrefix(contentType, "application/x-www-form-urlencoded;")) { // Check the content type
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

		// pull the parameters from the form
		text := r.FormValue("text")
		log.Printf("%s %s: text %d bytes\n", r.Method, r.URL.Path, len(text))
		removeBadBytes := cbIsSet(r.FormValue("remove-bad-bytes"))
		log.Printf("%s %s: removeBadBytes %v\n", r.Method, r.URL.Path, removeBadBytes)
		removeSensitiveLines := cbIsSet(r.FormValue("remove-sensitive-lines"))
		log.Printf("%s %s: removeSensitiveLines %v\n", r.Method, r.URL.Path, removeSensitiveLines)

		// convert eol on the input file
		var lines [][]byte
		if data := []byte(text); len(data) > 0 {
			data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
			data = bytes.ReplaceAll(data, []byte{'\r'}, []byte{'\n'})
			lines = bytes.Split(data, []byte{'\n'})
			for n, line := range lines {
				lines[n] = bytes.TrimLeft(line, " \t\r\n")
			}
			lines = trimLeadingBlankLines(trimTrailingBlankLines(lines))
		}
		log.Printf("%s %s: daFile: lines %d\n", r.Method, r.URL.Path, len(lines))

		// extract the clan and turn from the first two lines of the input
		var clanId, turnId string
		if len(lines) == 0 {
			http.Redirect(w, r, "/reports/uploads/failed?reason=input is empty", http.StatusSeeOther)
			return
		} else if len(lines) == 1 {
			http.Redirect(w, r, "/reports/uploads/failed?reason=input has only one line", http.StatusSeeOther)
			return
		} else if len(lines) == 2 {
			http.Redirect(w, r, "/reports/uploads/failed?reason=input has only two lines", http.StatusSeeOther)
			return
		} else if len(lines[0]) < 45 {
			http.Redirect(w, r, fmt.Sprintf("/reports/uploads/failed?reason=tribe header has only %d characters", len(lines[0])), http.StatusSeeOther)
			return
		} else if len(lines[0]) > 85 {
			http.Redirect(w, r, fmt.Sprintf("/reports/uploads/failed?reason=tribe header has %d characters", len(lines[0])), http.StatusSeeOther)
			return
		}
		clanFields := bytes.Split(lines[0], []byte{','})
		switch n := len(clanFields); n {
		case 0:
			http.Redirect(w, r, "/reports/uploads/failed?reason=tribe header contains 0 fields", http.StatusSeeOther)
			return
		case 1:
			http.Redirect(w, r, "/reports/uploads/failed?reason=tribe header contains only 1 field", http.StatusSeeOther)
			return
		case 2, 3:
			http.Redirect(w, r, fmt.Sprintf("/reports/uploads/failed?reason=tribe header contains only %d fields", len(clanFields)), http.StatusSeeOther)
			return
		case 4:
			// accept
		default:
			http.Redirect(w, r, fmt.Sprintf("/reports/uploads/failed?reason=tribe header contains %d fields", len(clanFields)), http.StatusSeeOther)
			return
		}
		tribeField, currHexField, prevHexField := clanFields[0], bytes.TrimSpace(clanFields[2]), bytes.TrimSpace(clanFields[3])
		if !bytes.HasPrefix(tribeField, []byte("Tribe 0")) {
			http.Redirect(w, r, "/reports/uploads/failed?reason=expected first line to start with \"Tribe 0\"", http.StatusSeeOther)
			return
		} else if !bytes.HasPrefix(currHexField, []byte("Current Hex = ")) {
			http.Redirect(w, r, "/reports/uploads/failed?reason=third field of tribe header is not current hex", http.StatusSeeOther)
			return
		} else if !bytes.HasPrefix(prevHexField, []byte("(Previous Hex = ")) {
			http.Redirect(w, r, "/reports/uploads/failed?reason=fourth field of tribe header is not previous hex", http.StatusSeeOther)
			return
		}
		if fields := strings.Fields(string(tribeField)); len(fields) != 2 {
			http.Redirect(w, r, "/reports/uploads/failed?reason=tribe header missing clan id", http.StatusSeeOther)
			return
		} else {
			clanId = fields[1]
			log.Printf("%s %s: clan fields: clanId %q\n", r.Method, r.URL.Path, clanId)
		}
		// split the second line into four fields
		turnFields := bytes.Split(lines[1], []byte{','})
		log.Printf("%s %s: turn %d\n", r.Method, r.URL.Path, len(turnFields))
		switch n := len(turnFields); n {
		case 0:
			http.Redirect(w, r, "/reports/uploads/failed?reason=current turn line is missing", http.StatusSeeOther)
			return
		case 1:
			http.Redirect(w, r, "/reports/uploads/failed?reason=current turn line has only 1 field", http.StatusSeeOther)
			return
		case 2, 3:
			http.Redirect(w, r, fmt.Sprintf("/reports/uploads/failed?reason=current turn line has only %d fields", len(turnFields)), http.StatusSeeOther)
			return
		case 4:
			// accept
		default:
			http.Redirect(w, r, fmt.Sprintf("/reports/uploads/failed?reason=current turn line has %d fields", len(turnFields)), http.StatusSeeOther)
			return
		}
		if !bytes.HasPrefix(turnFields[0], []byte("Current Turn ")) {
			log.Printf("%s %s: turn fields: invalid current turn\n", r.Method, r.URL.Path)
			http.Redirect(w, r, "/reports/uploads/failed?reason=second line does not start with \"Current Turn\"", http.StatusSeeOther)
			return
		}
		// the first field should be "Current Turn" space and a turn number
		currentTurnFields := strings.Fields(string(turnFields[0]))
		if len(currentTurnFields) != 4 {
			http.Redirect(w, r, "/reports/uploads/failed?reason=could not decipher current turn", http.StatusSeeOther)
			return
		} else {
			turnId = currentTurnFields[2]
			log.Printf("%s %s: turn fields: turnId %q\n", r.Method, r.URL.Path, turnId)
		}

		var fileName string
		if matches := rxTurnReports.FindStringSubmatch(turnId + "." + clanId + ".report.txt"); len(matches) != 4 {
			log.Printf("%s %s: matches %d\n", r.Method, r.URL.Path, len(matches))
			http.Redirect(w, r, "/reports/uploads/failed?reason=file name does not match clan and turn from header", http.StatusSeeOther)
			return
		} else {
			var year, month, clanId int
			if year, err = strconv.Atoi(matches[1]); err != nil {
				log.Printf("%s %s: year %v\n", r.Method, r.URL.Path, err)
				http.Redirect(w, r, "/reports/uploads/failed?reason=turn year is invalid", http.StatusSeeOther)
				return
			} else if year < 899 || year > 1234 {
				log.Printf("%s %s: year %d\n", r.Method, r.URL.Path, year)
				http.Redirect(w, r, "/reports/uploads/failed?reason=turn year is invalid", http.StatusSeeOther)
				return
			} else if month, err = strconv.Atoi(matches[2]); err != nil {
				log.Printf("%s %s: month %v\n", r.Method, r.URL.Path, err)
				http.Redirect(w, r, "/reports/uploads/failed?reason=turn month is invalid", http.StatusSeeOther)
				return
			} else if month < 1 || month > 12 {
				log.Printf("%s %s: month %d\n", r.Method, r.URL.Path, month)
				http.Redirect(w, r, "/reports/uploads/failed?reason=turn month is invalid", http.StatusSeeOther)
				return
			} else if clanId, err = strconv.Atoi(matches[3]); err != nil {
				log.Printf("%s %s: clan %v\n", r.Method, r.URL.Path, err)
				http.Redirect(w, r, "/reports/uploads/failed?reason=clan id is invalid", http.StatusSeeOther)
				return
			} else if clanId < 1 || clanId > 999 {
				log.Printf("%s %s: clan %d\n", r.Method, r.URL.Path, month)
				http.Redirect(w, r, "/reports/uploads/failed?reason=clan id is invalid", http.StatusSeeOther)
				return
			}
			fileName = fmt.Sprintf("%04d-%02d.%04d.report.txt", year, month, clanId)
		}
		log.Printf("%s %s: reportFileName %q\n", r.Method, r.URL.Path, fileName)

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

		reportFile := filepath.Join(inputPath, fileName)
		log.Printf("%s %s: creating %q\n", r.Method, r.URL.Path, reportFile)

		if removeSensitiveLines {
			lines = trimNonMappingLines(lines)
		}
		data := bytes.Join(lines, []byte{'\n'})
		if removeBadBytes {
			data = replaceInvalidUTF8(data)
		}
		if len(data) == 0 || data[len(data)-1] != '\n' {
			data = append(data, '\n')
		}
		if err := os.WriteFile(reportFile, data, 0644); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: created  %q\n", r.Method, r.URL.Path, reportFile)

		log.Printf("%s %s: wrote    %d bytes\n", r.Method, r.URL.Path, len(data))

		http.Redirect(w, r, "/reports/uploads/success?filename="+fileName, http.StatusSeeOther)
	}
}

func (s *Server) getApiVersionV1() http.HandlerFunc {
	buf, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf)
	}
}

func (s *Server) getCalendar(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "calendar", "content.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			if bytesWritten == 0 {
				log.Printf("%s %s: exited (%s)\n", r.Method, r.URL.Path, time.Since(started))
			} else {
				log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
			}
		}()

		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		user, err := s.extractSession(r)
		if err != nil {
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
			Heading: "Calendar",
			Content: calendar.Content{
				ClanId: user.Clan,
			},
			Footer: footer,
		}
		payload.CurrentPage.Calendar = true
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

func (s *Server) getDashboard(path string, footer app.Footer, cacheBuster bool) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "dashboard", "content.gohtml"),
		filepath.Join(path, "app", "pages", "dashboard", "turn-files-htmx.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			if bytesWritten == 0 {
				log.Printf("%s %s: exited (%s)\n", r.Method, r.URL.Path, time.Since(started))
			} else {
				log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
			}
		}()

		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Redirect(w, r, "/login?internal_server_error=true", http.StatusSeeOther)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Redirect(w, r, "/login?session_expired=true", http.StatusSeeOther)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		content := dashboard.Content{
			ClanId: user.Clan,
		}
		cf, err := s.stores.ffs.GetClanFiles(user)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Redirect(w, r, "/login?internal_server_error=true", http.StatusSeeOther)
			return
		}

		turns := map[string]*dashboard.TurnFiles_t{}
		for _, f := range cf.ErrorFiles {
			turn, ok := turns[f.Turn]
			if !ok {
				turn = &dashboard.TurnFiles_t{
					Turn:   f.Turn,
					ClanId: f.Clan,
				}
				turns[f.Turn] = turn
			}
			fi := &app.FileInfo_t{
				Owner: user.Clan,
				Name:  f.Name,
				Turn:  f.Turn,
				Clan:  f.Clan,
				Kind:  app.FIKError,
				Date:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02"),
				Time:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("15:04:05"),
				Route: fmt.Sprintf("/errlog/%s.%s", f.Turn, f.Clan),
				Path:  f.Path,
			}
			if cacheBuster {
				fi.Route += fmt.Sprintf("?ctl=%s", f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006.01.02.15.04.05"))
			}
			turn.Errors = append(turn.Errors, fi)
		}
		for _, f := range cf.LogFiles {
			turn, ok := turns[f.Turn]
			if !ok {
				turn = &dashboard.TurnFiles_t{
					Turn:   f.Turn,
					ClanId: f.Clan,
				}
				turns[f.Turn] = turn
			}
			fi := &app.FileInfo_t{
				Owner: user.Clan,
				Name:  f.Name,
				Turn:  f.Turn,
				Clan:  f.Clan,
				Kind:  app.FIKLog,
				Date:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02"),
				Time:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("15:04:05"),
				Route: fmt.Sprintf("/log/%s.%s", f.Turn, f.Clan),
				Path:  f.Path,
			}
			if cacheBuster {
				fi.Route += fmt.Sprintf("?ctl=%s", f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006.01.02.15.04.05"))
			}
			turn.Logs = append(turn.Logs, fi)
		}
		for _, f := range cf.MapFiles {
			turn, ok := turns[f.Turn]
			if !ok {
				turn = &dashboard.TurnFiles_t{
					Turn:   f.Turn,
					ClanId: f.Clan,
				}
				turns[f.Turn] = turn
			}
			fi := &app.FileInfo_t{
				Owner: user.Clan,
				Name:  f.Name,
				Turn:  f.Turn,
				Clan:  f.Clan,
				Kind:  app.FIKMap,
				Date:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02"),
				Time:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("15:04:05"),
				Route: fmt.Sprintf("/map/%s", f.Name),
				Path:  f.Path,
			}
			if cacheBuster {
				fi.Route += fmt.Sprintf("?ctl=%s", f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006.01.02.15.04.05"))
			}
			turn.Maps = append(turn.Maps, fi)
		}
		for _, f := range cf.ReportFiles {
			turn, ok := turns[f.Turn]
			if !ok {
				turn = &dashboard.TurnFiles_t{
					Turn:   f.Turn,
					ClanId: f.Clan,
				}
				turns[f.Turn] = turn
			}
			fi := &app.FileInfo_t{
				Owner: user.Clan,
				Name:  f.Name,
				Turn:  f.Turn,
				Clan:  f.Clan,
				Kind:  app.FIKReport,
				Date:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02"),
				Time:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("15:04:05"),
				Route: fmt.Sprintf("/report/%s", f.Name),
				Path:  f.Path,
			}
			if cacheBuster {
				fi.Route += fmt.Sprintf("?ctl=%s", f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006.01.02.15.04.05"))
			}
			turn.Reports = append(turn.Reports, fi)
		}
		for _, v := range turns {
			content.Turns = append(content.Turns, v)
		}
		sort.Slice(content.Turns, func(i, j int) bool {
			return content.Turns[i].Less(content.Turns[j])
		})

		payload := app.Layout{
			Title:   fmt.Sprintf("Clan %s", user.Clan),
			Heading: "Dashboard",
			Content: content,
			Footer:  footer,
		}
		payload.CurrentPage.Dashboard = true
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
	}
}

func (s *Server) deleteErrorLogLogId(components string) http.HandlerFunc {
	rxLogId, err := regexp.Compile(`^(\d{4})-(\d{2}).(\d{4})$`)
	if err != nil {
		log.Printf("error: deleteErrorLogLogId: %v\n", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

	files := []string{
		filepath.Join(components, "app", "pages", "dashboard", "turn-files-htmx.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)

		if r.Method != "DELETE" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		logId := r.PathValue("log_id")
		log.Printf("%s %s: log_id %q\n", r.Method, r.URL.Path, logId)
		matches := rxLogId.FindStringSubmatch(logId)
		log.Printf("%s %s: matches %+v\n", r.Method, r.URL.Path, matches)
		if len(matches) != 4 {
			log.Printf("%s %s: invalid log id: %d\n", r.Method, r.URL.Path, len(matches))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// validate every field of the log id
		var turnId string
		if year, err := strconv.Atoi(matches[1]); err != nil || year < 899 || year > 1380 {
			log.Printf("%s %s: invalid log id: year\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if month, err := strconv.Atoi(matches[2]); err != nil || month < 1 || month > 12 {
			log.Printf("%s %s: invalid log id: month\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if clan, err := strconv.Atoi(matches[3]); err != nil || clan < 1 || clan > 1000 {
			log.Printf("%s %s: invalid log id: clan\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			turnId = fmt.Sprintf("%04d-%02d", year, month)
		}
		if turnId == "" {
			log.Printf("%s %s: invalid log id: turn id\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// delete the file
		path := filepath.Join(user.Data, "logs", logId+".err")
		log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		if err := os.Remove(path); err != nil {
			// normally we would fail on an error, but we want to return the details to the user
			log.Printf("%s %s: r %v\n", r.Method, r.URL.Path, err)
		}

		// rebuild the turn details
		details, err := s.clanTurnFileList(user, turnId, s.features.cacheBuster)
		if err != nil {
			// normally we would fail on an error, but we want to return the details to the user
			log.Printf("%s %s: ctfl %v\n", r.Method, r.URL.Path, err)
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: parsed htmx components\n", r.Method, r.URL.Path)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.ExecuteTemplate(buf, "turn-files", details); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getErrorLogLogId() http.HandlerFunc {
	rxLogId, err := regexp.Compile(`^(\d{4})-(\d{2}).(\d{4})$`)
	if err != nil {
		log.Printf("error: getErrorLogLogId: %v\n", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		logId := r.PathValue("log_id")
		log.Printf("%s %s: log_id %q\n", r.Method, r.URL.Path, logId)
		matches := rxLogId.FindStringSubmatch(logId)
		log.Printf("%s %s: matches %+v\n", r.Method, r.URL.Path, matches)
		if len(matches) != 4 {
			log.Printf("%s %s: invalid log id: %d\n", r.Method, r.URL.Path, len(matches))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// validate every field of the log id
		if year, err := strconv.Atoi(matches[1]); err != nil || year < 899 || year > 1380 {
			log.Printf("%s %s: invalid log id: year\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if month, err := strconv.Atoi(matches[2]); err != nil || month < 1 || month > 12 {
			log.Printf("%s %s: invalid log id: month\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if clan, err := strconv.Atoi(matches[3]); err != nil || clan < 1 || clan > 1000 {
			log.Printf("%s %s: invalid log id: clan\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// does the file exist in the userdata directory?
		path := filepath.Join(user.Data, "logs", logId+".err")
		log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		sb, err := os.Stat(path)
		if err != nil || sb == nil || sb.IsDir() || !sb.Mode().IsRegular() {
			log.Printf("%s %s: invalid log id: file %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		bytesWritten = int(sb.Size())

		// serve the file
		http.ServeFile(w, r, path)
	}
}

func (s *Server) getHeroPage(path string, page string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "hero", "layout.gohtml"),
	}
	payload := hero.Layout{
		Title:   "OttoMap",
		Version: version.String(),
	}
	switch page {
	case "about":
		files = append(files, filepath.Join(path, "hero", "pages", "about-page.gohtml"))
	case "contact-us":
		files = append(files, filepath.Join(path, "hero", "pages", "contact-us-page.gohtml"))
	case "docs":
		files = append(files, filepath.Join(path, "hero", "pages", "docs-page.gohtml"))
	case "docs/converting-turn-reports":
		files = append(files, filepath.Join(path, "hero", "pages", "docs-converting-turn-reports-page.gohtml"))
	case "docs/dashboard-overview":
		files = append(files, filepath.Join(path, "hero", "pages", "docs-dashboard-overview-page.gohtml"))
	case "docs/errors":
		files = append(files, filepath.Join(path, "hero", "pages", "docs-errors-page.gohtml"))
	case "docs/getting-started":
		files = append(files, filepath.Join(path, "hero", "pages", "docs-getting-started-page.gohtml"))
	case "docs/map-key":
		files = append(files, filepath.Join(path, "hero", "pages", "docs-map-key.gohtml"))
	case "docs/ottomap-for-tribenet":
		files = append(files, filepath.Join(path, "hero", "pages", "docs-ottomap-for-tribenet-page.gohtml"))
	case "docs/report-layout":
		files = append(files, filepath.Join(path, "hero", "pages", "docs-report-layout.gohtml"))
	case "get-started":
		files = append(files, filepath.Join(path, "hero", "pages", "get-started-page.gohtml"))
	case "landing":
		files = append(files, filepath.Join(path, "hero", "pages", "landing-page.gohtml"))
	case "learn-more":
		files = append(files, filepath.Join(path, "hero", "pages", "learn-more-page.gohtml"))
	case "privacy":
		files = append(files, filepath.Join(path, "hero", "pages", "privacy-page.gohtml"))
	case "trusted":
		files = append(files, filepath.Join(path, "hero", "pages", "trusted-page.gohtml"))
	default:
		panic("!")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

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
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getIndex(serveStaticFiles bool, assets string, landing http.HandlerFunc) http.HandlerFunc {
	var assetsFS http.Handler
	if serveStaticFiles {
		assetsFS = http.FileServer(http.Dir(assets))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if assetsFS == nil && r.URL.Path != "/" {
			// production mode, nginx is handling static assets
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		started := time.Now()
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: exited (%s)\n", r.Method, r.URL.Path, time.Since(started))
		}()

		// development mode or nginx is not handling static assets, so serve them ourselves
		if r.URL.Path != "/" { // request has a path so it must be a request for an asset
			assetsFS.ServeHTTP(w, r)
			return
		}

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user != nil {
			// there is an active session, so redirect to dashboard
			log.Printf("%s %s: clan %q\n", r.Method, r.URL.Path, user.Clan)
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		// no session, so redirect to hero page
		landing(w, r)
	}
}

func (s *Server) deleteLogLogId(components string) http.HandlerFunc {
	rxLogId, err := regexp.Compile(`^(\d{4})-(\d{2}).(\d{4})$`)
	if err != nil {
		log.Printf("error: deleteLogLogId: %v\n", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

	files := []string{
		filepath.Join(components, "app", "pages", "dashboard", "turn-files-htmx.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)

		if r.Method != "DELETE" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		logId := r.PathValue("log_id")
		log.Printf("%s %s: log_id %q\n", r.Method, r.URL.Path, logId)
		matches := rxLogId.FindStringSubmatch(logId)
		log.Printf("%s %s: matches %+v\n", r.Method, r.URL.Path, matches)
		if len(matches) != 4 {
			log.Printf("%s %s: invalid log id: %d\n", r.Method, r.URL.Path, len(matches))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// validate every field of the log id
		var turnId string
		if year, err := strconv.Atoi(matches[1]); err != nil || year < 899 || year > 1380 {
			log.Printf("%s %s: invalid log id: year\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if month, err := strconv.Atoi(matches[2]); err != nil || month < 1 || month > 12 {
			log.Printf("%s %s: invalid log id: month\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if clan, err := strconv.Atoi(matches[3]); err != nil || clan < 1 || clan > 1000 {
			log.Printf("%s %s: invalid log id: clan\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			turnId = fmt.Sprintf("%04d-%02d", year, month)
		}
		if turnId == "" {
			log.Printf("%s %s: invalid log id: turn id\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// delete the file
		path := filepath.Join(user.Data, "logs", logId+".log")
		log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		if err := os.Remove(path); err != nil {
			// normally we would fail on an error, but we want to return the details to the user
			log.Printf("%s %s: r %v\n", r.Method, r.URL.Path, err)
		}

		// rebuild the turn details
		details, err := s.clanTurnFileList(user, turnId, s.features.cacheBuster)
		if err != nil {
			// normally we would fail on an error, but we want to return the details to the user
			log.Printf("%s %s: ctfl %v\n", r.Method, r.URL.Path, err)
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: parsed htmx components\n", r.Method, r.URL.Path)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.ExecuteTemplate(buf, "turn-files", details); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getLogLogId() http.HandlerFunc {
	rxLogId, err := regexp.Compile(`^(\d{4})-(\d{2}).(\d{4})$`)
	if err != nil {
		log.Printf("error: getLogLogId: %v\n", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		logId := r.PathValue("log_id")
		log.Printf("%s %s: log_id %q\n", r.Method, r.URL.Path, logId)
		matches := rxLogId.FindStringSubmatch(logId)
		log.Printf("%s %s: matches %+v\n", r.Method, r.URL.Path, matches)
		if len(matches) != 4 {
			log.Printf("%s %s: invalid log id: %d\n", r.Method, r.URL.Path, len(matches))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// validate every field of the log id
		if year, err := strconv.Atoi(matches[1]); err != nil || year < 899 || year > 1380 {
			log.Printf("%s %s: invalid log id: year\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if month, err := strconv.Atoi(matches[2]); err != nil || month < 1 || month > 12 {
			log.Printf("%s %s: invalid log id: month\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if clan, err := strconv.Atoi(matches[3]); err != nil || clan < 1 || clan > 1000 {
			log.Printf("%s %s: invalid log id: clan\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// does the file exist in the userdata directory?
		path := filepath.Join(user.Data, "logs", logId+".log")
		log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		sb, err := os.Stat(path)
		if err != nil || sb == nil || sb.IsDir() || !sb.Mode().IsRegular() {
			log.Printf("%s %s: invalid log id: file %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		bytesWritten = int(sb.Size())

		// serve the file
		http.ServeFile(w, r, path)
	}
}

func (s *Server) getLogin(path string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "pages", "login.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// redirect to the login clan page if they have the remember me cookie set
		cookie, err := r.Cookie(s.sessions.rememberMe)
		if err == nil && len(cookie.Value) == 4 {
			clanId := cookie.Value
			if n, err := strconv.Atoi(clanId); err == nil && (0 < n && n <= 999) {
				http.Redirect(w, r, fmt.Sprintf("/login/clan/%s", clanId), http.StatusSeeOther)
				return
			}
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: parsed components\n", r.Method, r.URL.Path)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.Execute(buf, nil); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getLoginClanId(path string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "pages", "login_clan.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// delete any existing session on the client
		if _, err := r.Cookie(s.sessions.cookieName); err == nil {
			log.Printf("%s %s: purging cookies\n", r.Method, r.URL.Path)
			http.SetCookie(w, &http.Cookie{
				Name:   s.sessions.cookieName,
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
		}

		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		var payload pages.LoginClan
		if clanId := r.PathValue("clan_id"); len(clanId) == 4 {
			if n, err := strconv.Atoi(clanId); err != nil || n < 0 || n > 999 {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			payload.ClanId, payload.Email = clanId, clanId+"@ottomap"
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.Execute(buf, payload); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) postLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login?invalid_credentials=true", http.StatusSeeOther)
		return
	}
}

func (s *Server) postLoginClanId() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// delete any existing session on the client
		if _, err := r.Cookie(s.sessions.cookieName); err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:   s.sessions.cookieName,
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
		}

		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		clanId := r.PathValue("clan_id")
		if len(clanId) != 4 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if n, err := strconv.Atoi(clanId); err != nil || n < 0 || n > 999 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// extract the email and password from the request
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		input := struct {
			email      string
			password   string
			rememberMe bool
		}{
			email:      r.Form.Get("email"),
			password:   r.Form.Get("password"),
			rememberMe: r.Form.Get("remember-me") == "on" || r.Form.Get("remember-me") == "true",
		}

		// clan id must match the email
		if input.email != clanId+"@ottomap" {
			http.Redirect(w, r, fmt.Sprintf("/login/clan/%s?invalid_credentials=true", clanId), http.StatusSeeOther)
			return
		}

		// check the password against the database
		user, err := s.stores.sessions.AuthenticateUser(input.email, input.password)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Redirect(w, r, fmt.Sprintf("/login/clan/%s?invalid_credentials=true", clanId), http.StatusSeeOther)
			return
		}
		loggedIn := user.Roles.IsActive

		// if the check fails, send them back to the login page
		if !loggedIn {
			http.Redirect(w, r, fmt.Sprintf("/login/clan/%s?invalid_credentials=true", clanId), http.StatusSeeOther)
			return
		}

		sessionId, err := s.stores.sessions.CreateSession(user.ID, s.sessions.ttl)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// set the session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     s.sessions.cookieName,
			Value:    sessionId,
			Path:     "/",
			MaxAge:   s.sessions.maxAge,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})
		if input.rememberMe {
			// set the clan tracking cookie
			http.SetCookie(w, &http.Cookie{
				Name:     s.sessions.rememberMe,
				Value:    clanId,
				Path:     "/",
				MaxAge:   6 * 30 * 24 * 60 * 60, // 6 months!
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
			})
		}

		// redirect to the dashboard
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

func (s *Server) getLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// we don't delete cookies if they are missing because we're afraid of leaking cookie names to bad clients

		// delete the session cookie if we have one
		if _, err := r.Cookie(s.sessions.cookieName); err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:   s.sessions.cookieName,
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
		}

		// delete the remember me cookie if we have one
		if _, err := r.Cookie(s.sessions.rememberMe); err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:   s.sessions.rememberMe,
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
		}

		// redirect to the landing page
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *Server) deleteMapMapId(components string) http.HandlerFunc {
	rxMap, err := regexp.Compile(`^(\d{4})-(\d{2}).(\d{4})\.wxx$`)
	if err != nil {
		log.Printf("error: deleteMapMapId: %v\n", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

	files := []string{
		filepath.Join(components, "app", "pages", "dashboard", "turn-files-htmx.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)

		if r.Method != "DELETE" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		mapId := r.PathValue("map_id")
		log.Printf("%s %s: log_id %q\n", r.Method, r.URL.Path, mapId)
		matches := rxMap.FindStringSubmatch(mapId)
		log.Printf("%s %s: matches %+v\n", r.Method, r.URL.Path, matches)
		if len(matches) != 4 {
			log.Printf("%s %s: invalid map id: %d\n", r.Method, r.URL.Path, len(matches))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// validate every field of the report id
		var turnId string
		if year, err := strconv.Atoi(matches[1]); err != nil || year < 899 || year > 1380 {
			log.Printf("%s %s: invalid map id: year\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if month, err := strconv.Atoi(matches[2]); err != nil || month < 1 || month > 12 {
			log.Printf("%s %s: invalid map id: month\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if clan, err := strconv.Atoi(matches[3]); err != nil || clan < 1 || clan > 1000 {
			log.Printf("%s %s: invalid map id: clan\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			turnId = fmt.Sprintf("%04d-%02d", year, month)
		}
		if turnId == "" {
			log.Printf("%s %s: invalid map id: turn id\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// delete the file
		path := filepath.Join(user.Data, "output", mapId)
		log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		if err := os.Remove(path); err != nil {
			// normally we would fail on an error, but we want to return the details to the user
			log.Printf("%s %s: r %v\n", r.Method, r.URL.Path, err)
		}

		// rebuild the turn details
		details, err := s.clanTurnFileList(user, turnId, s.features.cacheBuster)
		if err != nil {
			// normally we would fail on an error, but we want to return the details to the user
			log.Printf("%s %s: ctfl %v\n", r.Method, r.URL.Path, err)
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: parsed htmx components\n", r.Method, r.URL.Path)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.ExecuteTemplate(buf, "turn-files", details); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getMapMapId() http.HandlerFunc {
	rxMap, err := regexp.Compile(`^(\d{4})-(\d{2}).(\d{4})\.wxx$`)
	if err != nil {
		log.Printf("error: getMapMapId: %v\n", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		//log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		mapId := r.PathValue("map_id")
		log.Printf("%s %s: map_id %q\n", r.Method, r.URL.Path, mapId)
		matches := rxMap.FindStringSubmatch(mapId)
		//log.Printf("%s %s: matches %+v\n", r.Method, r.URL.Path, matches)
		if len(matches) != 4 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// validate every field of the map id
		if year, err := strconv.Atoi(matches[1]); err != nil || year < 899 || year > 1380 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if month, err := strconv.Atoi(matches[2]); err != nil || month < 1 || month > 12 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if clan, err := strconv.Atoi(matches[3]); err != nil || clan < 1 || clan > 1000 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// does the file exist in the userdata directory?
		path := filepath.Join(user.Data, "output", mapId)
		//log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		if sb, err := os.Stat(path); err != nil || sb.IsDir() || !sb.Mode().IsRegular() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			bytesWritten = int(sb.Size())
		}

		// jam in some headers to prevent issues with Windows + Edge
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", mapId))

		// serve the file
		http.ServeFile(w, r, path)
	}
}

func (s *Server) getMaps(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "maps", "content.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			if bytesWritten == 0 {
				log.Printf("%s %s: exited (%s)\n", r.Method, r.URL.Path, time.Since(started))
			} else {
				log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
			}
		}()

		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		user, err := s.extractSession(r)
		if err != nil {
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
			Heading: "Maps",
			Content: dashboard.Content{
				ClanId: user.Clan,
			},
			Footer: footer,
		}
		payload.CurrentPage.Maps = true
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

func (s *Server) deleteReportReportId(components string) http.HandlerFunc {
	rxReport, err := regexp.Compile(`^(\d{4})-(\d{2}).(\d{4})\.report.txt$`)
	if err != nil {
		log.Printf("error: deleteReportReportId: %v\n", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

	files := []string{
		filepath.Join(components, "app", "pages", "dashboard", "turn-files-htmx.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)

		if r.Method != "DELETE" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		reportId := r.PathValue("report_id")
		log.Printf("%s %s: log_id %q\n", r.Method, r.URL.Path, reportId)
		matches := rxReport.FindStringSubmatch(reportId)
		log.Printf("%s %s: matches %+v\n", r.Method, r.URL.Path, matches)
		if len(matches) != 4 {
			log.Printf("%s %s: invalid report id: %d\n", r.Method, r.URL.Path, len(matches))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// validate every field of the report id
		var turnId string
		if year, err := strconv.Atoi(matches[1]); err != nil || year < 899 || year > 1380 {
			log.Printf("%s %s: invalid report id: year\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if month, err := strconv.Atoi(matches[2]); err != nil || month < 1 || month > 12 {
			log.Printf("%s %s: invalid report id: month\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if clan, err := strconv.Atoi(matches[3]); err != nil || clan < 1 || clan > 1000 {
			log.Printf("%s %s: invalid report id: clan\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			turnId = fmt.Sprintf("%04d-%02d", year, month)
		}
		if turnId == "" {
			log.Printf("%s %s: invalid report id: turn id\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// delete the file
		path := filepath.Join(user.Data, "input", reportId)
		log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		if err := os.Remove(path); err != nil {
			// normally we would fail on an error, but we want to return the details to the user
			log.Printf("%s %s: r %v\n", r.Method, r.URL.Path, err)
		}

		// rebuild the turn details
		details, err := s.clanTurnFileList(user, turnId, s.features.cacheBuster)
		if err != nil {
			// normally we would fail on an error, but we want to return the details to the user
			log.Printf("%s %s: ctfl %v\n", r.Method, r.URL.Path, err)
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: parsed htmx components\n", r.Method, r.URL.Path)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.ExecuteTemplate(buf, "turn-files", details); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getReportBetaDocxToJson() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		user, err := s.extractSession(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		reportId := "docx-to-text.json"

		// does the file exist in the userdata directory?
		path := filepath.Join(user.Data, "input", reportId)
		if sb, err := os.Stat(path); err != nil || sb.IsDir() || !sb.Mode().IsRegular() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// serve the file
		http.ServeFile(w, r, path)
	}
}

func (s *Server) getReportBetaDocxToText() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		user, err := s.extractSession(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		//log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		reportId := "docx-to-text.txt"

		// does the file exist in the userdata directory?
		path := filepath.Join(user.Data, "input", reportId)
		//log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		if sb, err := os.Stat(path); err != nil || sb.IsDir() || !sb.Mode().IsRegular() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// serve the file
		http.ServeFile(w, r, path)
	}
}

func (s *Server) getReportReportId() http.HandlerFunc {
	rxReport, err := regexp.Compile(`^(\d{4})-(\d{2}).(\d{4})\.report.txt$`)
	if err != nil {
		log.Printf("error: getReportReportId: %v\n", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		started, bytesWritten := time.Now(), 0
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		defer func() {
			log.Printf("%s %s: wrote %d bytes in %s\n", r.Method, r.URL.Path, bytesWritten, time.Since(started))
		}()

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		//log.Printf("%s %s: session: clan_id %q\n", r.Method, r.URL.Path, user.Clan)

		reportId := r.PathValue("report_id")
		//log.Printf("%s %s: report_id %q\n", r.Method, r.URL.Path, reportId)
		matches := rxReport.FindStringSubmatch(reportId)
		log.Printf("%s %s: matches %+v\n", r.Method, r.URL.Path, matches)
		if len(matches) != 4 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// validate every field of the report id
		if year, err := strconv.Atoi(matches[1]); err != nil || year < 899 || year > 1380 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if month, err := strconv.Atoi(matches[2]); err != nil || month < 1 || month > 12 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if clan, err := strconv.Atoi(matches[3]); err != nil || clan < 1 || clan > 1000 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// does the file exist in the userdata directory?
		path := filepath.Join(user.Data, "input", reportId)
		//log.Printf("%s %s: path %q\n", r.Method, r.URL.Path, path)
		if sb, err := os.Stat(path); err != nil || sb.IsDir() || !sb.Mode().IsRegular() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			bytesWritten = int(sb.Size())
		}

		// serve the file
		http.ServeFile(w, r, path)
	}
}

func (s *Server) getReports(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "reports", "content.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
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

		// fetch the reports for the current user
		content := reports.Content_t{
			ClanId: user.Clan,
		}
		if cf, err := s.stores.ffs.GetClanFiles(user); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else {
			turns := map[string]*reports.Turn_t{}
			for _, f := range cf.LogFiles {
				turn, ok := turns[f.Turn]
				if !ok {
					turn = &reports.Turn_t{
						Turn:  f.Turn,
						Files: []reports.File_t{reports.File_t{}},
					}
					turns[f.Turn] = turn
				}
				turn.Files[0].ClanId = f.Clan
				turn.Files[0].Log = f.Name
			}
			for _, f := range cf.MapFiles {
				turn, ok := turns[f.Turn]
				if !ok {
					turn = &reports.Turn_t{
						Turn:  f.Turn,
						Files: []reports.File_t{reports.File_t{}},
					}
					turns[f.Turn] = turn
				}
				turn.Files[0].ClanId = f.Clan
				turn.Files[0].Map = f.Name
			}
			for _, f := range cf.ReportFiles {
				turn, ok := turns[f.Turn]
				if !ok {
					turn = &reports.Turn_t{
						Turn:  f.Turn,
						Files: []reports.File_t{reports.File_t{}},
					}
					turns[f.Turn] = turn
				}
				turn.Files[0].ClanId = f.Clan
				turn.Files[0].Report = f.Name
			}
			for _, v := range turns {
				content.Turns = append(content.Turns, v)
			}
		}

		payload := app.Layout{
			Title:   fmt.Sprintf("Clan %s", user.Clan),
			Heading: "Reports",
			Content: content,
			Footer:  footer,
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

func (s *Server) getReportsTurnIdClanId(path string) http.HandlerFunc {
	rxClanId := regexp.MustCompile(`^0[0-9]{3}$`)
	rxTurnId := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}$`)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		user, err := s.extractSession(r)
		if err != nil {
			log.Printf("%s %s: extractSession: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if user == nil {
			// there is no active session, so this is an error
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		// verify that we have an input directory for the clan
		inputPath := filepath.Join(user.Data, "input")
		if sb, err := os.Stat(inputPath); err != nil || !sb.IsDir() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		turnId, clanId := r.PathValue("turn_id"), r.PathValue("clan_id")
		if !rxClanId.MatchString(clanId) || !rxTurnId.MatchString(turnId) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// serve a report file if it exists
		for _, file := range []string{
			fmt.Sprintf("%s.%s.scrubbed.txt", turnId, clanId),
			fmt.Sprintf("%s.%s.report.txt", turnId, clanId),
		} {
			path := filepath.Join(inputPath, file)
			if sb, err := os.Stat(path); err != nil || sb.IsDir() || !sb.Mode().IsRegular() {
				continue
			}
			http.ServeFile(w, r, path)
			return
		}

		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
}

func (s *Server) getReportsUploads(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "reports", "uploads", "content.gohtml"),
		filepath.Join(path, "app", "widgets", "notifications.gohtml"),
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
			Content: uploads.Content_t{},
			Footer:  footer,
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

func (s *Server) getReportsUploadsFailed(path string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "pages", "reports", "failed", "content.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		payload := app.Layout{
			Title:   "Upload Failed",
			Heading: "Reports",
			Content: failed.Content_t{
				Reason: r.URL.Query().Get("reason"),
			},
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.Execute(buf, payload); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getReportsUploadsSuccess(path string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "pages", "reports", "success", "content.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.Execute(buf, nil); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getSettings(path string, footer app.Footer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/settings/general", http.StatusSeeOther)
	}
}

func (s *Server) getSettingsGeneral(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "pages", "settings", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "settings", "general", "content.gohtml"),
		filepath.Join(path, "app", "pages", "settings", "general", "timezone-htmx.gohtml"),
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

		// trim the user data path for the template
		var userData = "***user data not found***"
		if index := strings.Index(user.Data, "/"+user.Clan); index != -1 {
			userData = user.Data[index:]
		}

		payload := settings.Layout_t{
			Title:  fmt.Sprintf("Clan %s", user.Clan),
			ClanId: user.Clan,
			Footer: footer,
		}
		payload.CurrentPage.General = true
		payload.Footer.Timestamp = time.Now().In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02 15:04:05")
		content := general.Content_t{
			ClanId:      user.Clan,
			AccountName: user.Clan + "@ottomap",
			Roles:       "Chief",
			Data:        userData,
		}
		if user.Roles.IsActive {
			content.Roles += ", active"
		}
		if user.Roles.IsAdministrator {
			content.Roles += ", adminstrator"
		}
		if user.Roles.IsAuthenticated {
			content.Roles += ", authenticated"
		}
		if user.Roles.IsOperator {
			content.Roles += ", operator"
		}
		if user.Roles.IsUser {
			content.Roles += ", user"
		}
		content.LanguageAndDates.DateFormat = "YYYY-MM-DD"
		content.LanguageAndDates.Timezone.Name = user.LanguageAndDates.Timezone.Location.String()
		content.LanguageAndDates.TimezoneSelect = general.TimezoneSelectList(user.LanguageAndDates.Timezone.Location)
		payload.Content = content

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

func (s *Server) getSettingsGeneralTimezone(path string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "pages", "settings", "general", "timezone-htmx.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		if r.Method != "GET" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
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

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: parsed htmx components\n", r.Method, r.URL.Path)

		payload := general.TimezoneSelectList(user.LanguageAndDates.Timezone.Location)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.ExecuteTemplate(buf, "timezone", payload); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) postSettingsGeneralTimezone(path string) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "pages", "settings", "general", "timezone-htmx.gohtml"),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s: entered\n", r.Method, r.URL.Path)
		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		} else if r.Header.Get("HX-Request") != "true" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if contentType := r.Header.Get("Content-Type"); !(contentType == "application/x-www-form-urlencoded" || strings.HasPrefix(contentType, "application/x-www-form-urlencoded;")) {
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

		// pull the parameters from the form
		newLocation := r.FormValue("timezone-location")
		if newLocation == "" {
			newLocation = "UTC"
		}
		log.Printf("%s %s: newLocation %q\n", r.Method, r.URL.Path, newLocation)
		loc, err := time.LoadLocation(newLocation)
		if loc == nil {
			log.Printf("%s %s: invalid timezone %s\n", r.Method, r.URL.Path, newLocation)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		} else if err := s.stores.store.UpdateUserTimezone(user.ID, loc); err != nil {
			log.Printf("%s %s: updateUserTimezone: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		t, err := template.ParseFiles(files...)
		if err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("%s %s: parsed htmx components\n", r.Method, r.URL.Path)

		payload := general.TimezoneSelectList(loc)

		// parse into a buffer so that we can handle errors without writing to the response
		buf := &bytes.Buffer{}
		if err := t.ExecuteTemplate(buf, "timezone", payload); err != nil {
			log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) getSettingsPlans(path string, footer app.Footer) http.HandlerFunc {
	files := []string{
		filepath.Join(path, "app", "pages", "settings", "layout.gohtml"),
		filepath.Join(path, "app", "pages", "settings", "plans", "content.gohtml"),
	}

	content := plans.Content
	for n := range content.Cards {
		content.Cards[n].Id = n + 1
		content.Cards[n].TopRow = n < 2
		content.Cards[n].LeftColumn = n%2 == 0
		content.Cards[n].RightColumn = n%2 == 1
	}
	if len(content.Cards) > 0 {
		content.Cards[len(content.Cards)-1].BottomRow = true
		if content.Cards[len(content.Cards)-1].RightColumn {
			content.Cards[len(content.Cards)-2].BottomRow = false
		}
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

		payload := settings.Layout_t{
			Title:   fmt.Sprintf("Clan %s", user.Clan),
			ClanId:  user.Clan,
			Content: content,
			Footer:  footer,
		}
		payload.CurrentPage.Plans = true
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

func (s *Server) clanTurnFileList(user *domains.User_t, turnId string, cacheBuster bool) (*dashboard.TurnFiles_t, error) {
	turn := &dashboard.TurnFiles_t{
		Turn:   turnId,
		ClanId: user.Clan,
	}

	cf, err := s.stores.ffs.GetClanFiles(user)
	if err != nil {
		return turn, err
	}

	for _, f := range cf.ErrorFiles {
		if f.Turn != turnId {
			continue
		}
		fi := &app.FileInfo_t{
			Owner: user.Clan,
			Name:  f.Name,
			Turn:  f.Turn,
			Clan:  f.Clan,
			Kind:  app.FIKError,
			Date:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02"),
			Time:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("15:04:05"),
			Route: fmt.Sprintf("/errlog/%s.%s", f.Turn, f.Clan),
			Path:  f.Path,
		}
		if cacheBuster {
			fi.Route += fmt.Sprintf("?ctl=%s", f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006.01.02.15.04.05"))
		}
		turn.Errors = append(turn.Errors, fi)
	}

	for _, f := range cf.LogFiles {
		if f.Turn != turnId {
			continue
		}
		fi := &app.FileInfo_t{
			Owner: user.Clan,
			Name:  f.Name,
			Turn:  f.Turn,
			Clan:  f.Clan,
			Kind:  app.FIKLog,
			Date:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02"),
			Time:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("15:04:05"),
			Route: fmt.Sprintf("/log/%s.%s", f.Turn, f.Clan),
			Path:  f.Path,
		}
		if cacheBuster {
			fi.Route += fmt.Sprintf("?ctl=%s", f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006.01.02.15.04.05"))
		}
		turn.Logs = append(turn.Logs, fi)
	}

	for _, f := range cf.MapFiles {
		if f.Turn != turnId {
			continue
		}
		fi := &app.FileInfo_t{
			Owner: user.Clan,
			Name:  f.Name,
			Turn:  f.Turn,
			Clan:  f.Clan,
			Kind:  app.FIKMap,
			Date:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02"),
			Time:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("15:04:05"),
			Route: fmt.Sprintf("/map/%s", f.Name),
			Path:  f.Path,
		}
		if cacheBuster {
			fi.Route += fmt.Sprintf("?ctl=%s", f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006.01.02.15.04.05"))
		}
		turn.Maps = append(turn.Maps, fi)
	}

	for _, f := range cf.ReportFiles {
		if f.Turn != turnId {
			continue
		}
		fi := &app.FileInfo_t{
			Owner: user.Clan,
			Name:  f.Name,
			Turn:  f.Turn,
			Clan:  f.Clan,
			Kind:  app.FIKReport,
			Date:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006-01-02"),
			Time:  f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("15:04:05"),
			Route: fmt.Sprintf("/report/%s", f.Name),
			Path:  f.Path,
		}
		if cacheBuster {
			fi.Route += fmt.Sprintf("?ctl=%s", f.Timestamp.In(user.LanguageAndDates.Timezone.Location).Format("2006.01.02.15.04.05"))
		}
		turn.Reports = append(turn.Reports, fi)
	}

	turn.IsEmpty = turn.Errors == nil && turn.Logs == nil && turn.Maps == nil && turn.Reports == nil

	return turn, nil
}
