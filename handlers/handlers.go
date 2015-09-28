// Package handlers is where the HTTP server work is done.
package handlers

// TODO: Move error handlers to error package

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/dustin/go-humanize"
	"github.com/therealbill/airbrake-go"
)

var TemplateBase string
var STATIC_URL string

// PageContext holds all the contextual information a page will want to return,
// use, or display
type PageContext struct {
	Title        string
	SubTitle     string
	Data         interface{}
	Static       string
	ViewTemplate string
	CurrentURL   string
	Refresh      bool
	RefreshTime  int
	RefreshURL   string
	Error        error
}

// NewPageContext instantiates and returns a PageContext with "global" data
// already set.
func NewPageContext() (pc PageContext, err error) {
	// any default initialization common to all pages goes here
	pc = PageContext{Static: STATIC_URL}
	return
}

// getTemplateList returns the base template and the requested template
func getTemplateList(tname string) []string {
	base := TemplateBase + "html/templates/base.html"
	thisOne := TemplateBase + "html/templates/" + tname + ".html"
	tmpl_list := []string{base, thisOne}
	return tmpl_list
}

// HumanizeBigBytes transforms a uint64 to a human readable string such as
// "100Kb"
func HumanizeBigBytes(bytes int64) string {
	res := humanize.Bytes(uint64(bytes))
	return res
}

// HumanizeBytes transforms an int to a human readable string such as
// "100Kb"
func HumanizeBytes(bytes int) string {
	res := humanize.Bytes(uint64(bytes))
	return res
}

// CommifyFloat turns a float into a string with comma separation
func CommifyFloat(bytes float64) string {
	res := humanize.Comma(int64(bytes))
	return res
}

// IntFromFloat64 provides a convenience function fo convert an int to a float
// insert screed about how you probably should not do it but sometimes you need
// to here.
func IntFromFloat64(incoming float64) (i int) {
	i = int(incoming)
	return i
}

// Turn an "ok" string into a boolean
func OkToBool(ok string) bool {
	if ok == "ok" {
		return true
	}
	return false
}

// render is called to turn the processed data into a rendered page in a
// handelr function. This way we can add features such as authentication to a
// central rendering call rather than go through each web page function and do
// it there.
func render(w http.ResponseWriter, context PageContext) {
	funcMap := template.FuncMap{
		"title":            strings.Title,
		"HumanizeBytes":    HumanizeBytes,
		"HumanizeBigBytes": HumanizeBigBytes,
		"CommifyFloat":     CommifyFloat,
		"Float2Int":        IntFromFloat64,
		"OkToBool":         OkToBool,
		"tableflip":        func() string { return "(╯°□°）╯︵ ┻━┻" },
	}
	context.Static = STATIC_URL
	tmpl_list := getTemplateList(context.ViewTemplate)
	/*
		t, err := template.ParseFiles(tmpl_list...)
		if err != nil {
			log.Print("template parsing error: ", err)
		}
	*/
	t := template.Must(template.New("base.html").Funcs(funcMap).ParseFiles(tmpl_list...))
	err := t.Execute(w, context)
	if err != nil {
		log.Print("template executing error: ", err)
	}
}

// throwJSONParseError is used when the JSON a client submits via the API isn't
// parseable
func throwJSONParseError(req *http.Request) (retcode int, userMessage string) {
	retcode = 422
	userMessage = "JSON Parse failure"
	em := fmt.Errorf(userMessage)
	e := airbrake.ExtendedNotification{ErrorClass: "Request.ParseJSON", Error: em}
	err := airbrake.ExtendedError(e, req)
	if err != nil {
		log.Print("airbrake error:", err)
	}
	return
}

//checkContextError is used to valiate we received a valid PageContext back
//from the call, returning an error i, re, reqqf not
func checkContextError(err error, w *http.ResponseWriter) (retcode int, userMessage string) {
	if err != nil {
		log.Printf("Context error: %s", err.Error())
		http.Error(*w, "Context not initialized. See server log for details", http.StatusInternalServerError)
		return 500, "Server Context Error"
	}
	return 200, ""
}

//returnUnhandledError is used to valiate we received a valid PageContext back
//from the call, returning an error i, re, reqqf not
func returnUnhandledError(err error, w *http.ResponseWriter) (doReturn bool) {
	if err != nil {
		log.Printf("Unhandled error: %s", err.Error())
		http.Error(*w, "Error not handled. See server log for details", http.StatusInternalServerError)
		return true
	}
	return false
}
