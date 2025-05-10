// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
)

func (s *Server) renderFragment(payload any, templateName string, templateFiles ...string) ([]byte, error) {
	t, err := template.ParseFiles(templateFiles...)
	if err != nil {
		log.Printf("%s: %v\n", templateName, err)
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := t.ExecuteTemplate(buf, templateName, payload); err != nil {
		log.Printf("%s: %v\n", templateName, err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *Server) writeFragments(w http.ResponseWriter, r *http.Request, fragments ...[]byte) (bytesWritten int, err error) {
	// create a buffer so that we can handle errors without writing to the response
	buf := &bytes.Buffer{}
	for _, fragment := range fragments {
		if _, err = buf.Write(fragment); err != nil {
			break
		}
	}

	if err != nil {
		log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return 0, err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	return w.Write(buf.Bytes())

}

func (s *Server) writeHtmxFragment(w http.ResponseWriter, r *http.Request, payload any, templateName string, templateFiles ...string) (int, error) {
	t, err := template.ParseFiles(templateFiles...)
	if err != nil {
		log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return 0, err
	}
	log.Printf("%s %s: parsed components\n", r.Method, r.URL.Path)

	// parse into a buffer so that we can handle errors without writing to the response
	buf := &bytes.Buffer{}
	if err := t.ExecuteTemplate(buf, templateName, payload); err != nil {
		log.Printf("%s %s: %v\n", r.Method, r.URL.Path, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return 0, err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	return w.Write(buf.Bytes())
}
