package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/fatih/color"
	hornbillpasswordgen "github.com/hornbill/goHornbillPasswordGen"
)

func processRegexOnString(reg string, input string) string {
	re1, err := regexp.Compile(reg)
	if err != nil {
		logger(4, "Regex Error: "+fmt.Sprintf("%v", err), false)
		return ""
	}
	//-- Get Array of all Matched max 100
	result := re1.FindAllString(input, 100)
	strReturn := ""
	//-- Loop Matches
	for _, match := range result {
		strReturn = match

		if strReturn != "" {
			return strReturn
		}
	}

	return strReturn
}

func getUserFieldValue(u *map[string]interface{}, s string) string {
	//-- Dynamically Grab Mapped Value
	r := reflect.ValueOf(googleImportConf.User.AccountMapping)
	f := reflect.Indirect(r).FieldByName(s)
	//-- Get Mapped Value
	var UserMapping = f.String()
	var stringToReturn = processComplexField(u, UserMapping)
	return stringToReturn
}

//-- Get XMLMC Feild from mapping via profile Object
func getProfileFieldValue(u *map[string]interface{}, s string) string {
	//-- Dyniamicly Grab Mapped Value
	r := reflect.ValueOf(googleImportConf.User.ProfileMapping)

	f := reflect.Indirect(r).FieldByName(s)

	//-- Get Mapped Value
	var UserProfileMapping = f.String()
	var stringToReturn = processComplexField(u, UserProfileMapping)
	return stringToReturn
}

//-- Match any value wrapped in [] and get its Google Attribute Value
func processComplexField(u *map[string]interface{}, s string) (value string) {
	t := template.New(s).Funcs(TemplateFilters)
	t, _ = t.Parse(s)
	buf := bytes.NewBufferString("")
	t.Execute(buf, u)
	if buf != nil {
		value = buf.String()
		if value == "<no value>" {
			value = ""
		}
	}
	return
}

//-- Generate Password String
func generatePasswordString(importData *userWorkingDataStruct) string {
	pwdinst := hornbillpasswordgen.NewPasswordInstance()
	pwdinst.Length = passwordProfile.Length
	pwdinst.UseLower = true
	pwdinst.ForceLower = passwordProfile.ForceLower
	pwdinst.UseNumeric = true
	pwdinst.ForceNumeric = passwordProfile.ForceNumeric
	pwdinst.UseUpper = true
	pwdinst.ForceUpper = passwordProfile.ForceUpper
	pwdinst.UseSpecial = true
	pwdinst.ForceSpecial = passwordProfile.ForceSpecial
	pwdinst.Blacklist = passwordProfile.Blacklist
	if passwordProfile.CheckMustNotContain {
		pwdinst.MustNotContain = append(pwdinst.MustNotContain, importData.Account.FirstName)
		pwdinst.MustNotContain = append(pwdinst.MustNotContain, importData.Account.LastName)
		pwdinst.MustNotContain = append(pwdinst.MustNotContain, importData.Account.UserID)
	}

	//Generate a new password
	newPassword, _, err := pwdinst.GenPassword()

	if err != nil {
		logger(4, "Failed Password Auto Generation for: "+importData.Account.UserID+"  "+fmt.Sprintf("%v", err), false)
		return ""
	}
	return newPassword
}

func loggerGen(t int, s string) string {
	//-- Ignore Logging level unless is 0
	if t < googleImportConf.Advanced.LogLevel && t != 0 {
		return ""
	}

	var errorLogPrefix = ""
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 3:
		errorLogPrefix = "[WARN] "
	case 4:
		errorLogPrefix = "[ERROR] "
	}
	return errorLogPrefix + s + "\n\r"
}
func loggerWriteBuffer(s string) {
	if s != "" {
		logLines := strings.Split(s, "\n\r")
		for _, line := range logLines {
			if line != "" {
				logger(0, line, false)
			}
		}
	}
}
func deletefiles(path string, f os.FileInfo, err error) (e error) {
	var cutoff = (24 * time.Hour)
	cutoff = time.Duration(googleImportConf.Advanced.LogRetention) * cutoff
	now := time.Now()
	// check each file if starts with prefix and our log name so other log files are not deleted and different imports can have differnt retentions
	if strings.HasPrefix(f.Name(), Flags.configLogPrefix+"Google_User_Import_") {

		if diff := now.Sub(f.ModTime()); diff > cutoff {
			logger(1, "Removing Old Log File: "+path, false)
			os.Remove(path)
		}

	}
	return

}

func runLogRetentionCheck() {
	logger(1, "Processing Old Log Files Current Retention Set to: "+fmt.Sprintf("%d", googleImportConf.Advanced.LogRetention), true)

	if googleImportConf.Advanced.LogRetention > 0 {
		//-- Curreny WD
		cwd, _ := os.Getwd()
		//-- Log Folder
		logPath := cwd + "/log"
		// walk through the files in the given path and perform partialrename()
		// function
		filepath.Walk(logPath, deletefiles)
	}

}

//-- Logging function
func logger(t int, s string, outputtoCLI bool) {

	//-- Ignore Logging level unless is 0
	if !Flags.configDebug && t == 1 {
		return
	}
	mutexLog.Lock()
	defer mutexLog.Unlock()

	onceLog.Do(func() {
		//-- Curreny WD
		cwd, _ := os.Getwd()
		//-- Log Folder
		logPath := cwd + "/log"
		//-- Log File
		logFileName := logPath + "/" + Flags.configLogPrefix + "Google_User_Import_" + Time.timeNow + ".log"
		//-- If Folder Does Not Exist then create it
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			err := os.Mkdir(logPath, 0777)
			if err != nil {
				fmt.Printf("Error Creating Log Folder %q: %s \r", logPath, err)
				os.Exit(101)
			}
		}

		//-- Open Log File
		var err error
		f, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			fmt.Printf("Error Creating Log File %q: %s \n", logFileName, err)
			os.Exit(100)
		}
		log.SetOutput(f)

	})
	// don't forget to close it
	//defer f.Close()
	red := color.New(color.FgRed).PrintfFunc()
	orange := color.New(color.FgCyan).PrintfFunc()
	var errorLogPrefix = ""
	//-- Create Log Entry
	switch t {
	case 0:
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 3:
		errorLogPrefix = "[WARN] "
	case 4:
		errorLogPrefix = "[ERROR] "
	}
	if outputtoCLI {
		if t == 3 {
			orange(errorLogPrefix + s + "\n")
		} else if t == 4 {
			red(errorLogPrefix + s + "\n")
		} else {
			fmt.Printf(errorLogPrefix + s + "\n")
		}

	}
	log.Println(errorLogPrefix + s)
}

func sysOptionGet(sysOption string) (optionValue string) {
	loggerAPI.SetParam("filter", sysOption)
	response, err := loggerAPI.Invoke("admin", "sysOptionGet")
	if err != nil {
		logger(4, "Could not retrieve System Setting ["+sysOption+"]: "+err.Error(), false)
		return
	}
	var jsonRespon xmlmcSettingResponse
	err = json.Unmarshal([]byte(response), &jsonRespon)
	if err != nil {
		logger(4, "Could not retrieve System Setting ["+sysOption+"]: "+err.Error(), false)
		return
	}
	if !jsonRespon.MethodResult {
		logger(4, "Could not retrieve System Setting ["+sysOption+"]: "+jsonRespon.State.Error, false)
		return
	}
	return jsonRespon.Params.Option[0].Value
}

func printOnly(r rune) rune {
	if unicode.IsPrint(r) {
		return r
	}
	return -1
}
