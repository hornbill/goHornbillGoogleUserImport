package main

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

func getGoogleUsers() {
	logger(2, "Querying Google user data", true)
	var nextPageToken string

	gEspXmlmc = apiLib.NewXmlmcInstance(Flags.configInstanceID)
	gEspXmlmc.SetAPIKey(Flags.configAPIKey)
	for {
		userList, err := getUsersPage(nextPageToken)
		if err != nil {
			os.Exit(1)
		}
		for _, v := range userList.Params.Data.Users {
			userDetails := make(map[string]interface{})
			for key, val := range v {
				switch key {
				case "addresses":
					for _, address := range val.([]interface{}) {

						addDetails := address.(map[string]interface{})
						userDetails[addDetails["type"].(string)+"Address"] = addDetails["formatted"].(string)
					}
				case "externalIds":
					for _, externalIds := range val.([]interface{}) {
						extIdDetails := externalIds.(map[string]interface{})
						if extIdDetails["type"].(string) == "organization" {
							if extIdDetails["organization"] != nil {
								userDetails["employeeId"] = extIdDetails["organization"].(string)
							}
						}
					}
				case "gender":
					userDetails["gender"] = val.(map[string]interface{})["type"]
				case "languages":
					for _, language := range val.([]interface{}) {
						langDetails := language.(map[string]interface{})
						if langDetails["preference"].(string) == "preferred" {
							userDetails["languageCode"] = langDetails["languageCode"].(string)
						}
					}
				case "locations":
					for _, location := range val.([]interface{}) {
						locDetails := location.(map[string]interface{})
						for lock, locv := range locDetails {
							if lock != "type" && lock != "area" {
								userDetails[lock] = locv.(string)
							}
						}
					}
				case "name":
					userDetails["familyName"] = val.(map[string]interface{})["familyName"]
					userDetails["givenName"] = val.(map[string]interface{})["givenName"]
					userDetails["fullName"] = val.(map[string]interface{})["fullName"]
				case "organizations":
					for _, organization := range val.([]interface{}) {
						orgDetails := organization.(map[string]interface{})
						for orgk, orgv := range orgDetails {
							if orgk == "title" {
								orgk = "jobTitle"
							}
							if orgk == "description" {
								orgk = "employeeType"
							}
							switch orgvVal := orgv.(type) {
							default:
								userDetails[orgk] = orgvVal
							}
						}
					}
				case "phones":
					for _, phone := range val.([]interface{}) {
						phoneDetails := phone.(map[string]interface{})
						userDetails[phoneDetails["type"].(string)+"Phone"] = phoneDetails["value"].(string)
					}
				case "relations":
					for _, relation := range val.([]interface{}) {
						relDetails := relation.(map[string]interface{})
						if relDetails["type"].(string) == "manager" {
							userDetails["managerEmail"] = relDetails["value"].(string)
						}
					}
				default:
					userDetails[key] = val
				}
			}
			localGoogleUsers = append(localGoogleUsers, userDetails)
		}
		// Google's API will return a token even when on the last page of data.
		// So break the loop if no token is returned
		if userList.Params.Data.NextPageToken == "" {
			break
		}
		nextPageToken = userList.Params.Data.NextPageToken
	}
	if len(localGoogleUsers) == 0 {
		logger(3, "No user records returned from Google - check your configuration!", true)
		os.Exit(0)
	}
	logger(2, "Total user records returned from Google: "+strconv.Itoa(len(localGoogleUsers)), true)
}

func getUsersPage(pageToken string) (usersResponse googleResponseStruct, err error) {
	var payload = usersPayloadStruct{
		Customer:    googleImportConf.GoogleConf.Customer,
		Domain:      googleImportConf.GoogleConf.Domain,
		MaxResults:  maxGoogleResults,
		PageToken:   pageToken,
		Query:       googleImportConf.GoogleConf.Query,
		ShowDeleted: false,
	}

	strPayload, err := json.Marshal(payload)
	if err != nil {
		logger(4, "getUsersPage::marshal:Error parsing request payload:"+err.Error(), true)
		return
	}
	gEspXmlmc.SetParam("methodPath", "/Google/Workspace/DataSources.system/List Users.m")
	gEspXmlmc.SetParam("requestPayload", string(strPayload))
	gEspXmlmc.OpenElement("credential")
	gEspXmlmc.SetParam("id", "googleworkspace")
	gEspXmlmc.SetParam("keyId", strconv.Itoa(googleImportConf.GoogleConf.KeysafeID))
	gEspXmlmc.CloseElement("credential")

	requestPayloadXML := gEspXmlmc.GetParam()
	responsePayloadXML, err := gEspXmlmc.Invoke("bpm", "iBridgeInvoke")
	if err != nil {
		logger(4, "getUsersPage::iBridgeInvoke:invoke:"+err.Error(), true)
		logger(4, "Request XML: "+requestPayloadXML, false)
		return
	}
	var xmlRespon xmlmcIBridgeResponse
	err = xml.Unmarshal([]byte(strings.Map(printOnly, string(responsePayloadXML))), &xmlRespon)
	if err != nil {
		logger(4, "getUsersPage::iBridgeInvoke:unmarshal:"+err.Error(), true)
		logger(4, "Request XML: "+requestPayloadXML, false)
		logger(4, "Response XML: "+responsePayloadXML, false)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "getUsersPage::iBridgeInvoke:methodResult:"+xmlRespon.State.Error, true)
		logger(4, "Request XML: "+requestPayloadXML, false)
		logger(4, "Response XML: "+responsePayloadXML, false)
		return
	}
	if xmlRespon.IBridgeResponseError != "" {
		logger(4, "getUsersPage::iBridgeInvoke:responseError:"+xmlRespon.IBridgeResponseError, true)
		logger(4, "Request XML: "+requestPayloadXML, false)
		logger(4, "Response XML: "+responsePayloadXML, false)
		return
	}

	err = json.Unmarshal([]byte(xmlRespon.IBridgeResponsePayload), &usersResponse)
	if err != nil {
		logger(4, "getUsersPage::iBridgeInvoke:jsonUnmarshal:"+err.Error(), true)
		logger(4, "JSON: "+xmlRespon.IBridgeResponsePayload, false)
	}
	return
}

func getUsersImage(userKey string) (imageData googleImageStruct, err error) {
	var payload = usersImagePayloadStruct{
		UserKey: userKey,
	}

	strPayload, err := json.Marshal(payload)
	if err != nil {
		return
	}
	gEspXmlmc.SetParam("methodPath", "/Google/Workspace/DataSources.system/Get User Image.m")
	gEspXmlmc.SetParam("requestPayload", string(strPayload))
	gEspXmlmc.OpenElement("credential")
	gEspXmlmc.SetParam("id", "googleworkspace")
	gEspXmlmc.SetParam("keyId", strconv.Itoa(googleImportConf.GoogleConf.KeysafeID))
	gEspXmlmc.CloseElement("credential")
	requestPayloadXML := gEspXmlmc.GetParam()
	responsePayloadXML, err := gEspXmlmc.Invoke("bpm", "iBridgeInvoke")
	if err != nil {
		logger(4, "getUsersImage::iBridgeInvoke:invoke:"+err.Error(), true)
		logger(4, "Request XML: "+requestPayloadXML, true)
		return
	}
	var xmlRespon xmlmcIBridgeResponse
	err = xml.Unmarshal([]byte(strings.Map(printOnly, string(responsePayloadXML))), &xmlRespon)
	if err != nil {
		logger(4, "getUsersImage::iBridgeInvoke:unmarshal:"+err.Error(), true)
		logger(4, "Request XML: "+requestPayloadXML, true)
		logger(4, "Response XML: "+responsePayloadXML, true)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "getUsersImage::iBridgeInvoke:methodResult:"+xmlRespon.State.Error, true)
		logger(4, "Request XML: "+requestPayloadXML, true)
		logger(4, "Response XML: "+responsePayloadXML, true)
		return
	}
	var usersResponse googleResponseStruct
	err = json.Unmarshal([]byte(xmlRespon.IBridgeResponsePayload), &usersResponse)
	if err != nil {
		logger(4, "getUsersImage::iBridgeInvoke:jsonUnmarshal:"+err.Error(), true)
		logger(4, "JSON: "+xmlRespon.IBridgeResponsePayload, false)
	}

	imageData.Height = usersResponse.Params.Data.ImgHeight
	imageData.Width = usersResponse.Params.Data.ImgWidth
	imageData.MimeType = usersResponse.Params.Data.ImgMimeType
	imageData.PhotoData = processPhotoData(usersResponse.Params.Data.ImgPhotoData)
	return
}

func processPhotoData(websafeB64Image string) (cleanedImage string) {
	cleanedImage = strings.ReplaceAll(websafeB64Image, "_", "/")
	cleanedImage = strings.ReplaceAll(cleanedImage, "-", "+")
	cleanedImage = strings.ReplaceAll(cleanedImage, "*", "=")
	cleanedImage = strings.ReplaceAll(cleanedImage, ".", "=")
	return cleanedImage
}
