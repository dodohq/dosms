package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/schema"
)

// RenderJSON return json object in the http response
func RenderJSON(w http.ResponseWriter, response []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

// ReadRequestBody read request body and bind to and interface
func ReadRequestBody(r *http.Request, i interface{}) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/x-www-form-urlencoded" {
		if err := r.ParseForm(); err != nil {
			return err
		}

		decoder := schema.NewDecoder()
		decoder.IgnoreUnknownKeys(true)
		if err := decoder.Decode(i, r.PostForm); err != nil {
			return err
		}
	} else if contentType == "application/json" {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(i); err != nil {
			return err
		}
	} else if strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return err
		}
		decoder := schema.NewDecoder()
		if err := decoder.Decode(i, r.PostForm); err != nil {
			return err
		}
	} else {
		return errors.New("Content-Type Not Accepted")
	}
	return nil
}

// ReadFileUpload process csv upload and return the uploaded destination
func ReadFIleUpload(r *http.Request, fileFieldName string) (string, error) {
	file, _, err := r.FormFile(fileFieldName)
	defer file.Close()
	if err != nil {
		return "", err
	}

	filePath := "./tmp/" + strconv.FormatInt(time.Now().Unix(), 10)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	defer f.Close()
	if err != nil {
		return "", err
	}
	io.Copy(f, file)
	return filePath, nil
}
