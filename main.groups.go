package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

func getOrgFromLookup(l *userWorkingDataStruct, orgValue string, orgType int) string {
	//-- Check if Org Attribute is set
	if orgValue == "" {
		logger(1, "Org Lookup is Enabled but Attribute is not Defined", false)
		return ""
	}
	//-- Get Value of Attribute
	logger(1, "Google Attribute for Org Lookup: "+orgValue, false)
	orgAttributeName := processComplexField(&l.DB, orgValue)
	if orgAttributeName != "" {
		logger(1, "Looking Up Org "+orgAttributeName, false)

		//-- See if Group is cached
		found := false
		orgLookupID := ""
		orgLookupName := ""
		for _, v := range HornbillCache.GroupsID {
			if strings.EqualFold(orgAttributeName, v.Name) && orgType == v.Type {
				found = true
				orgLookupID = v.ID
				orgLookupName = v.Name
				break
			}
		}

		if found {
			logger(1, "Organisation Lookup found ID "+orgLookupName, false)
			return orgLookupID
		}
		logger(1, "Unable to Find Organisation "+orgAttributeName+" of type "+strconv.Itoa(orgType), false)
	}
	return ""
}

func userGroupsUpdate(hIF *apiLib.XmlmcInstStruct, user *userWorkingDataStruct, buffer *bytes.Buffer) (bool, error) {

	for groupIndex := range user.Groups {
		group := user.Groups[groupIndex]
		buffer.WriteString(loggerGen(1, "Group Add User: "+user.Account.CheckID+" ("+user.Jobs.id+")"+" Group: "+group.Name))

		//hIF.SetParam("userId", user.Account.UserID)
		hIF.SetParam("userId", user.Jobs.id)
		hIF.SetParam("groupId", group.ID)
		hIF.SetParam("memberRole", group.Membership)
		hIF.OpenElement("options")
		hIF.SetParam("tasksView", strconv.FormatBool(group.TasksView))
		hIF.SetParam("tasksAction", strconv.FormatBool(group.TasksAction))
		hIF.CloseElement("options")
		var XMLSTRING = hIF.GetParam()
		if Flags.configDryRun {
			buffer.WriteString(loggerGen(1, "Group Add User XML "+XMLSTRING))
			hIF.ClearParam()
			return true, nil
		}

		RespBody, xmlmcErr := hIF.Invoke("admin", "userAddGroup")
		var JSONResp xmlmcResponse
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(1, "Group Add User XML "+XMLSTRING))
			return false, xmlmcErr
		}
		err := json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			buffer.WriteString(loggerGen(1, "Group Add User XML "+XMLSTRING))
			return false, err
		}
		if JSONResp.State.Error != "" {
			buffer.WriteString(loggerGen(1, "Group Add User XML "+XMLSTRING))
			return false, errors.New(JSONResp.State.Error)
		}
		buffer.WriteString(loggerGen(1, "Group added to User: "+user.Account.CheckID+" ("+user.Jobs.id+")"))
	}

	return true, nil
}
func userGroupsRemove(hIF *apiLib.XmlmcInstStruct, user *userWorkingDataStruct, buffer *bytes.Buffer) (bool, error) {

	for groupIndex := range user.GroupsToRemove {
		group := user.GroupsToRemove[groupIndex]
		buffer.WriteString(loggerGen(1, "Group Remove User: "+user.Account.CheckID+" ("+user.Jobs.id+")"+" Group Id: "+group))

		hIF.SetParam("userId", user.Jobs.id)
		hIF.SetParam("groupId", group)

		var XMLSTRING = hIF.GetParam()
		if Flags.configDryRun {
			buffer.WriteString(loggerGen(1, "Group Remove User XML "+XMLSTRING))
			hIF.ClearParam()
			return true, nil
		}

		RespBody, xmlmcErr := hIF.Invoke("admin", "userDeleteGroup")
		var JSONResp xmlmcResponse
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(1, "Group Remove User XML "+XMLSTRING))
			return false, xmlmcErr
		}
		err := json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			buffer.WriteString(loggerGen(1, "Group Remove User XML "+XMLSTRING))
			return false, err
		}
		if JSONResp.State.Error != "" {
			buffer.WriteString(loggerGen(1, "Group Remove User XML "+XMLSTRING))
			return false, errors.New(JSONResp.State.Error)
		}
		buffer.WriteString(loggerGen(1, "Group Removed From User: "+user.Account.CheckID+" ("+user.Jobs.id+")"))
	}

	return true, nil
}

func userGroupSetHomeOrg(hIF *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) error {
	if currentUser.Account.HomeOrg == "" {
		err := "No Home Organisation set for User [" + currentUser.Account.CheckID + " (" + currentUser.Jobs.id + ")" + "]"
		buffer.WriteString(loggerGen(1, err))
		return errors.New(err)
	}
	hIF.SetParam("userId", currentUser.Jobs.id)
	//hIF.SetParam("userId", currentUser.Account.UserID)
	hIF.SetParam("homeOrganization", currentUser.Account.HomeOrg)
	XMLSTRING := hIF.GetParam()
	RespBody, xmlmcErr := hIF.Invoke("admin", "userUpdate")
	var JSONResp xmlmcResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(1, "User Set Home Org XML "+XMLSTRING))
		return xmlmcErr
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		buffer.WriteString(loggerGen(1, "User Set Home Org XML "+XMLSTRING))
		return err
	}
	if JSONResp.State.Error != "" {
		buffer.WriteString(loggerGen(1, "User Set Home Org XML "+XMLSTRING))
		return errors.New(JSONResp.State.Error)
	}
	buffer.WriteString(loggerGen(1, "Home Organisation ["+currentUser.Account.HomeOrg+"] set for User ["+currentUser.Account.CheckID+" ("+currentUser.Jobs.id+")"+"]"))
	return nil
}
