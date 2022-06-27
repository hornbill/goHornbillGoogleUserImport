package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	apiLib "github.com/hornbill/goApiLib"
)

func userImageUpdate(hIF *apiLib.XmlmcInstStruct, user *userWorkingDataStruct, buffer *bytes.Buffer) (bool, error) {
	//- Profile Images are already B64 encoded in cache
	buffer.WriteString(loggerGen(1, "User Profile Image Set: "+user.Account.UserID))
	value := ""

	relLink := "session/" + user.Account.UserID + "." + googleImportConf.User.Image.ImageType
	strDAVurl := hIF.DavEndpoint + relLink

	buffer.WriteString(loggerGen(1, "DAV Upload URL: "+strDAVurl))

	if !Flags.configDryRun {

		if user.Image.PhotoData != "" {
			putbody := base64.NewDecoder(base64.StdEncoding, strings.NewReader(user.Image.PhotoData))
			req, Perr := http.NewRequest("PUT", strDAVurl, putbody)
			if Perr != nil {
				return false, Perr
			}
			req.Header.Set("Content-Type", user.Image.MimeType)
			req.Header.Add("Authorization", "ESP-APIKEY "+Flags.configAPIKey)
			req.Header.Set("User-Agent", "Go-http-client/1.1")

			duration := time.Second * time.Duration(Flags.configAPITimeout)
			client := &http.Client{Timeout: duration}

			response, Perr := client.Do(req)
			if Perr != nil {
				return false, Perr
			}
			defer response.Body.Close()
			_, _ = io.Copy(ioutil.Discard, response.Body)
			if response.StatusCode == 201 || response.StatusCode == 200 {
				value = "/" + relLink
			}
		} else {
			buffer.WriteString(loggerGen(1, "Unable to Upload Profile Image to DAV as its empty"))
			return false, nil
		}

		buffer.WriteString(loggerGen(1, "Profile Set Image URL: "+value))
		hIF.SetParam("objectRef", "urn:sys:user:"+user.Account.UserID)
		hIF.SetParam("sourceImage", value)
		var XMLSTRING = hIF.GetParam()

		if Flags.configDryRun {
			buffer.WriteString(loggerGen(1, "Profile Image Set XML "+XMLSTRING))
			hIF.ClearParam()
			return false, nil
		}

		RespBody, xmlmcErr := hIF.Invoke("activity", "profileImageSet")
		var JSONResp xmlmcResponse
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(1, "Profile Image Set XML "+XMLSTRING))
			return false, xmlmcErr
		}
		err := json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			buffer.WriteString(loggerGen(1, "Profile Image Set XML "+XMLSTRING))
			return false, err
		}
		if JSONResp.State.Error != "" {
			buffer.WriteString(loggerGen(1, "Profile Image Set XML "+XMLSTRING))
			return false, errors.New(JSONResp.State.Error)
		}
		buffer.WriteString(loggerGen(1, "Image added to User: "+user.Account.UserID))

		//Now go delete the file from dav

		if user.Image.PhotoData != "" {
			reqDel, DelErr := http.NewRequest("DELETE", strDAVurl, nil)
			if DelErr != nil {
				buffer.WriteString(loggerGen(3, "User image updated but could not remove from session. Error: "+fmt.Sprintf("%v", DelErr)))
				return true, nil
			}
			reqDel.Header.Add("Authorization", "ESP-APIKEY "+Flags.configAPIKey)
			reqDel.Header.Set("User-Agent", "Go-http-client/1.1")

			duration := time.Second * time.Duration(Flags.configAPITimeout)
			client := &http.Client{Timeout: duration}

			responseDel, DelErr := client.Do(reqDel)
			if DelErr != nil {
				buffer.WriteString(loggerGen(3, "User image updated but could not remove from session. Error: "+fmt.Sprintf("%v", DelErr)))
				return true, nil
			}
			defer responseDel.Body.Close()
			_, _ = io.Copy(ioutil.Discard, responseDel.Body)
			if responseDel.StatusCode < 200 || responseDel.StatusCode > 299 {
				buffer.WriteString(loggerGen(3, "User image updated but could not remove from session. Status Code: "+strconv.Itoa(responseDel.StatusCode)))
			}
		}

	} else {
		buffer.WriteString(loggerGen(1, "[DRYRUN]"))
	}
	return true, nil
}
