package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	TemplateFilters template.FuncMap
)

func checkTemplate() bool {
	blnFoundError := false
	X := reflect.ValueOf(googleImportConf.User.AccountMapping)
	for i := 0; i < X.NumField(); i++ {
		str := X.Field(i).Interface().(string)
		t := template.New(str).Funcs(TemplateFilters)
		_, err := t.Parse(str)
		if err != nil {
			fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [AccountMapping." + str + "]")
			blnFoundError = true
		}
	}
	X = reflect.ValueOf(googleImportConf.User.ProfileMapping)
	for i := 0; i < X.NumField(); i++ {
		str := X.Field(i).Interface().(string)
		t := template.New(str).Funcs(TemplateFilters)
		_, err := t.Parse(str)
		if err != nil {
			fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [ProfileMapping." + str + "]")
			blnFoundError = true
		}
	}

	for _, Org := range googleImportConf.User.Org {
		str := Org.Value
		t := template.New(str).Funcs(TemplateFilters)
		_, err := t.Parse(str)
		if err != nil {
			fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [Org: " + Org.Value + "]")
			blnFoundError = true
		}
	}
	str := fmt.Sprintf("%v", googleImportConf.User.Site.Value)
	t := template.New(str).Funcs(TemplateFilters)
	_, err := t.Parse(str)
	if err != nil {
		fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [Site " + googleImportConf.User.Site.Value + "]")
		blnFoundError = true
	}
	str = fmt.Sprintf("%v", googleImportConf.User.Image.URI)
	t = template.New(str).Funcs(TemplateFilters)
	_, err = t.Parse(str)
	if err != nil {
		fmt.Println("[TEMPLATE] Parsing Error: " + err.Error() + " [Img " + googleImportConf.User.Image.URI + "]")
		blnFoundError = true
	}
	return blnFoundError
}

func setTemplateFilters() {
	TemplateFilters = template.FuncMap{
		"Upper": func(feature string) string {
			return strings.ToUpper(feature)
		},
		"Lower": func(feature string) string {
			return strings.ToLower(feature)
		},
		"epoch": func(feature string) string {
			result := ""
			if feature == "" {
			} else if feature == "0" {
			} else {
				t, err := strconv.ParseInt(feature, 10, 0)
				if err == nil {
					md := time.Unix(t, 0)
					result = md.Format("2006-01-02 15:04:05")
				}
			}
			return result
		},
		"epoch_clear": func(feature string) string {
			result := "__clear__"
			if feature == "" {
			} else if feature == "0" {
			} else {
				t, err := strconv.ParseInt(feature, 10, 0)
				if err == nil {
					md := time.Unix(t, 0)
					result = md.Format("2006-01-02 15:04:05")
				}
			}
			return result
		},
	}
}
