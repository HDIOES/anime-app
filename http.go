package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func logRequest(request *http.Request) (io.Reader, error) {
	logStringBuilder := new(strings.Builder)
	logStringBuilder.WriteString("Http request:\n")
	logStringBuilder.WriteString("Method ")
	logStringBuilder.WriteString(request.Method)
	logStringBuilder.WriteString(" ")
	logStringBuilder.WriteString(request.URL.String())
	logStringBuilder.WriteString("\n")
	data, readErr := ioutil.ReadAll(request.Body)
	if readErr != nil {
		return nil, errors.WithStack(readErr)
	}
	logStringBuilder.Write(data)
	logStringBuilder.WriteString("\n")
	log.Println(logStringBuilder)
	return bytes.NewBuffer(data), nil
}

func logResponse(response *http.Response) error {
	logStringBuilder := new(strings.Builder)
	logStringBuilder.WriteString("Http response:\n")
	logStringBuilder.WriteString("Http status: ")
	logStringBuilder.WriteString(strconv.Itoa(response.StatusCode))
	logStringBuilder.WriteString("\n")
	data, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return errors.WithStack(readErr)
	}
	logStringBuilder.Write(data)
	logStringBuilder.WriteString("\n")
	log.Print(logStringBuilder)
	return nil
}
