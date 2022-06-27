package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

//-- Store Google Users in Map
func processGoogleUsers() {
	logger(2, "Processing Google user records", true)

	//-- User Working Data
	HornbillCache.UsersWorking = make(map[string]*userWorkingDataStruct)
	//-- Loop Google Users
	for _, user := range localGoogleUsers {
		// Process Pre Import Actions
		var userID = processImportActions(&user)
		// Process Params and return userId
		processUserParams(&user, userID)
	}

	logger(2, "Google user records processed: "+fmt.Sprintf("%d", len(localGoogleUsers))+"\n", true)
}

func processData() {
	logger(2, "Processing user records", true)
	boolSkip := false
	for user := range HornbillCache.UsersWorking {
		boolSkip = false
		currentUser := HornbillCache.UsersWorking[user] //source user
		//-- Current UserID

		userExists := false

		var hornbillUserData userAccountStruct
		if checkHornbillUserData, ok := HornbillCache.Users[strings.ToLower(currentUser.Account.CheckID)]; ok {
			userExists = true
			hornbillUserData = checkHornbillUserData
		}

		userID := strings.ToLower(currentUser.Account.UserID)
		//-- Extra Debugging
		if userExists {
			logger(1, "Google User ID: '"+userID+"'", false)
		} else {
			logger(1, "Google User ID: '"+userID+"' NOT Found", false)
		}
		//-- Check Map no need to loop
		if userExists && strings.ToLower(googleImportConf.User.Operation) != "create" {
			currentUser.Jobs.id = hornbillUserData.HUserID
			logger(1, "Actual Account ID"+currentUser.Jobs.id+"'\n", false)
			currentUser.Jobs.update = checkUserNeedsUpdate(currentUser, hornbillUserData)
			currentUser.Jobs.updateProfile = checkUserNeedsProfileUpdate(currentUser, hornbillUserData)
			currentUser.Jobs.updateType = checkUserNeedsTypeUpdate(currentUser, hornbillUserData)
			currentUser.Jobs.updateSite = checkUserNeedsSiteUpdate(currentUser, hornbillUserData)
			currentUser.Jobs.updateImage = checkUserNeedsImageUpdate(currentUser, hornbillUserData)
			currentUser.Jobs.updateHomeOrg = checkUserNeedsHomeOrgUpdate(currentUser, hornbillUserData)
			checkUserNeedsOrgUpdate(currentUser, hornbillUserData)
			checkUserNeedsOrgRemoving(currentUser, hornbillUserData)
			checkUserNeedsRoleUpdate(currentUser, hornbillUserData)
			currentUser.Jobs.updateStatus = checkUserNeedsStatusUpdate(currentUser, hornbillUserData)
		} else if strings.ToLower(googleImportConf.User.Operation) != "update" && userID != "" {
			currentUser.Jobs.id = userID
			setUserPasswordValueForCreate(currentUser)
			setUserSiteValueForCreate(currentUser, hornbillUserData)
			setUserRolesValueForCreate(currentUser, hornbillUserData)
			currentUser.Jobs.updateImage = checkUserNeedsImageCreate(currentUser, hornbillUserData)
			checkUserNeedsOrgCreate(currentUser, hornbillUserData)
			currentUser.Jobs.updateStatus = checkUserNeedsStatusCreate(currentUser, hornbillUserData)
			currentUser.Jobs.updateHomeOrg = checkUserNeedsHomeOrgCreate(currentUser, hornbillUserData)
			currentUser.Jobs.updateProfile = checkUserNeedsProfileUpdate(currentUser, hornbillUserData)
			currentUser.Jobs.create = true
		} else {
			currentUser.Jobs.update = false
			currentUser.Jobs.updateProfile = false
			currentUser.Jobs.updateType = false
			currentUser.Jobs.updateSite = false
			currentUser.Jobs.updateImage = false
			currentUser.Jobs.updateHomeOrg = false
			currentUser.Jobs.create = false
			currentUser.Jobs.updateStatus = false
			currentUser.Jobs.updateHomeOrg = false
			boolSkip = true
		}

		if boolSkip {
			logger(1, userID+" will be skipped because of Action: "+googleImportConf.User.Operation+"\n", false)
			CounterInc(7)
			logger(4, "Google record has no User ID: '"+fmt.Sprintf("%+v", currentUser.DB)+"'\n", false)
		} else {
			loggerOutput := []string{
				"User: " + userID,
				"Operation: " + googleImportConf.User.Operation,
				"Create: " + strconv.FormatBool(currentUser.Jobs.create),
				"Update: " + strconv.FormatBool(currentUser.Jobs.update),
				"Update Type: " + strconv.FormatBool(currentUser.Jobs.updateType),
				"Update Profile: " + strconv.FormatBool(currentUser.Jobs.updateProfile),
				"Update Site: " + strconv.FormatBool(currentUser.Jobs.updateSite),
				"Update Status: " + strconv.FormatBool(currentUser.Jobs.updateStatus),
				"Update Home Organisation: " + strconv.FormatBool(currentUser.Jobs.updateHomeOrg),
				"Roles Count: " + fmt.Sprintf("%d", len(currentUser.Roles)),
				"Update Image: " + strconv.FormatBool(currentUser.Jobs.updateImage),
				"Groups: " + fmt.Sprintf("%d", len(currentUser.Groups))}
			strings.Join(loggerOutput[:], "\n\t")
			logger(1, strings.Join(loggerOutput[:], "\n\t")+"\n", false)
		}

	}
	logger(1, "User records processed: "+fmt.Sprintf("%d", len(HornbillCache.UsersWorking))+"", true)
}

func checkUserNeedsStatusCreate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if googleImportConf.User.Role.Action == "Both" || googleImportConf.User.Role.Action == "Create" {
		//-- By default, user records are created active so if we need to change the status it should be done if not active
		if googleImportConf.User.Status.Value != "active" {
			return true
		}
	}
	return false
}

func checkUserNeedsStatusUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if googleImportConf.User.Status.Action == "Both" || googleImportConf.User.Status.Action == "Update" {
		//-- Check current status != config status
		if HornbillUserStatusMap[currentData.HAccountStatus] != googleImportConf.User.Status.Value {
			return true
		}
	}
	return false
}

func setUserPasswordValueForCreate(importData *userWorkingDataStruct) {
	if importData.Account.Password == "" {
		//-- Generate Password
		importData.Account.Password = generatePasswordString(importData)
		logger(1, "Auto Generated Password for: "+importData.Account.UserID+" - "+importData.Account.Password, false)
	}
	//-- Base64 Encode
	importData.Account.Password = base64.StdEncoding.EncodeToString([]byte(importData.Account.Password))
}

func checkUserNeedsOrgRemoving(importData *userWorkingDataStruct, currentData userAccountStruct) {
	//-- Only if we have some config for groups
	if len(googleImportConf.User.Org) > 0 {

		//-- List of Existing Groups
		var userExistingGroups = HornbillCache.UserGroups[strings.ToLower(importData.Account.UserID)]
		for index := range userExistingGroups {
			ExistingGroupID := userExistingGroups[index]
			ExistingGroup := HornbillCache.GroupsID[strings.ToLower(ExistingGroupID)]
			boolGroupNeedsRemoving := false

			//-- Loop Config Orgs and Check each one
			for orgIndex := range googleImportConf.User.Org {

				//-- Get Group from Index
				importOrg := googleImportConf.User.Org[orgIndex]

				//-- Only if Actions is correct
				if importOrg.Action == "Both" || importOrg.Action == "Update" {
					//-- Evaluate the Id
					var GroupID = getOrgFromLookup(importData, importOrg.Value, orgTypes[importOrg.Options.Type])
					//-- If already a member of import group then ignore
					if GroupID == ExistingGroup.ID {
						//-- exit for loop
						continue
					}

					//-- If group we are a memember of matches the Type of a group we have set up on the import and its set to one Assignment
					if orgTypes[importOrg.Options.Type] == ExistingGroup.Type && importOrg.Options.OnlyOneGroupAssignment {
						boolGroupNeedsRemoving = true
					}
				}
			}
			//-- If group is not part of import and its set to remove
			if boolGroupNeedsRemoving {
				importData.GroupsToRemove = append(importData.GroupsToRemove, ExistingGroupID)
			}
		}
	}
}

func checkUserNeedsOrgUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) {
	if len(googleImportConf.User.Org) > 0 {
		for _, orgDetails := range googleImportConf.User.Org {
			orgAction := orgDetails
			if orgAction.Action == "Both" || orgAction.Action == "Update" {
				var GroupID = getOrgFromLookup(importData, orgAction.Value, orgTypes[orgAction.Options.Type])
				var userExistingGroups = HornbillCache.UserGroups[strings.ToLower(importData.Account.UserID)]
				//-- Is User Already a Memeber of the Group
				boolUserInGroup := false
				for index := range userExistingGroups {
					if strings.EqualFold(GroupID, userExistingGroups[index]) {
						boolUserInGroup = true
					}
				}

				if !boolUserInGroup && GroupID != "" {
					var group userGroupStruct
					group.ID = GroupID
					group.Name = orgAction.Value
					group.Type = orgTypes[orgAction.Options.Type]
					group.Membership = orgAction.Options.Membership
					group.TasksView = orgAction.Options.TasksView
					group.TasksAction = orgAction.Options.TasksAction
					group.OnlyOneGroupAssignment = orgAction.Options.OnlyOneGroupAssignment

					importData.Groups = append(importData.Groups, group)
				}
			}
		}
	}
}

func checkUserNeedsHomeOrgUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if len(googleImportConf.User.Org) > 0 {
		for orgIndex := range googleImportConf.User.Org {
			orgAction := googleImportConf.User.Org[orgIndex]
			if !orgAction.Options.SetAsHomeOrganisation {
				continue
			}
			if orgAction.Action == "Both" || orgAction.Action == "Update" {
				var GroupID = getOrgFromLookup(importData, orgAction.Value, orgTypes[orgAction.Options.Type])

				if GroupID == "" || strings.EqualFold(currentData.HHomeOrg, GroupID) {
					return false
				}
				importData.Account.HomeOrg = GroupID
				logger(1, "Home Organisation: "+GroupID+" - "+currentData.HHomeOrg, true)
				return true
			}
		}
	}
	return false
}

func checkUserNeedsHomeOrgCreate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if len(googleImportConf.User.Org) > 0 {
		for orgIndex := range googleImportConf.User.Org {
			orgAction := googleImportConf.User.Org[orgIndex]
			if !orgAction.Options.SetAsHomeOrganisation {
				continue
			}
			if orgAction.Action == "Create" || orgAction.Action == "Both" {
				var GroupID = getOrgFromLookup(importData, orgAction.Value, orgTypes[orgAction.Options.Type])

				if GroupID == "" || strings.EqualFold(currentData.HHomeOrg, GroupID) {
					return false
				}
				importData.Account.HomeOrg = GroupID
				logger(1, "Home Organisation: "+GroupID+" - "+currentData.HHomeOrg, true)
				return true
			}
		}
	}
	return false
}

func checkUserNeedsOrgCreate(importData *userWorkingDataStruct, currentData userAccountStruct) {
	if len(googleImportConf.User.Org) > 0 {
		for orgIndex := range googleImportConf.User.Org {
			orgAction := googleImportConf.User.Org[orgIndex]
			if orgAction.Action == "Both" || orgAction.Action == "Create" {

				var GroupID = getOrgFromLookup(importData, orgAction.Value, orgTypes[orgAction.Options.Type])
				var group userGroupStruct
				group.ID = GroupID
				group.Name = orgAction.Value
				group.Type = orgTypes[orgAction.Options.Type]
				group.Membership = orgAction.Options.Membership
				group.TasksView = orgAction.Options.TasksView
				group.TasksAction = orgAction.Options.TasksAction
				group.OnlyOneGroupAssignment = orgAction.Options.OnlyOneGroupAssignment

				if GroupID != "" {
					importData.Groups = append(importData.Groups, group)
				}
			}
		}
	}
}
func setUserRolesValueForCreate(importData *userWorkingDataStruct, currentData userAccountStruct) {
	if googleImportConf.User.Role.Action == "Both" || googleImportConf.User.Role.Action == "Create" {
		importData.Roles = googleImportConf.User.Role.Roles
	}
}
func checkUserNeedsRoleUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) {
	if googleImportConf.User.Role.Action == "Both" || googleImportConf.User.Role.Action == "Update" {
		for index := range googleImportConf.User.Role.Roles {
			roleName := googleImportConf.User.Role.Roles[index]
			foundRole := false
			var userRoles = HornbillCache.UserRoles[strings.ToLower(importData.Account.UserID)]
			for index2 := range userRoles {
				if strings.EqualFold(roleName, userRoles[index2]) {
					foundRole = true
				}
			}
			if !foundRole {
				importData.Roles = append(importData.Roles, roleName)
			}
		}
	}
}
func checkUserNeedsImageCreate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if googleImportConf.User.Image.Action == "Both" || googleImportConf.User.Image.Action == "Create" {
		image, err := getUsersImage(importData.DB["id"].(string))
		if err == nil {
			importData.Image = image
			return true

		}
	}
	return false
}
func checkUserNeedsImageUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if googleImportConf.User.Image.Action == "Both" || googleImportConf.User.Image.Action == "Update" {
		image, err := getUsersImage(importData.DB["id"].(string))
		if err == nil {
			importData.Image = image
			return true

		}
	}
	return false
}
func checkUserNeedsTypeUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if googleImportConf.User.Type.Action == "Both" || googleImportConf.User.Type.Action == "Update" {
		// -- 1 = user
		// -- 3 = basic
		switch importData.Account.UserType {
		case "user":
			if currentData.HClass != "1" {
				return true
			}
		case "basic":
			if currentData.HClass != "3" {
				return true
			}
		default:
			return false
		}
	} else {
		if currentData.HClass == "1" {
			importData.Account.UserType = "user"
		} else {
			importData.Account.UserType = "basic"
		}
	}
	return false
}
func setUserSiteValueForCreate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	//-- Is Site Enables for Update or both
	if googleImportConf.User.Site.Action == "Both" || googleImportConf.User.Site.Action == "Create" {
		importData.Account.Site = getSiteFromLookup(importData)
	}
	if importData.Account.Site != "" && importData.Account.Site != currentData.HSite {
		return true
	}
	return false
}
func checkUserNeedsSiteUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	//-- Is Site Enables for Update or both
	if googleImportConf.User.Site.Action == "Both" || googleImportConf.User.Site.Action == "Update" {
		importData.Account.Site = getSiteFromLookup(importData)
	} else {
		//-- Else Default to current value
		importData.Account.Site = currentData.HSite
	}

	if importData.Account.Site != "" && importData.Account.Site != currentData.HSite {
		return true
	}
	return false
}
func checkUserNeedsUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if importData.Account.LoginID != "" && importData.Account.LoginID != currentData.HLoginID {
		logger(1, "LoginID: "+importData.Account.LoginID+" - "+currentData.HLoginID, false)
		return true
	}
	if importData.Account.EmployeeID != "" && importData.Account.EmployeeID != currentData.HEmployeeID {
		logger(1, "EmployeeID: "+importData.Account.EmployeeID+" - "+currentData.HEmployeeID, false)
		return true
	}
	if importData.Account.Name != "" && importData.Account.Name != currentData.HName {
		logger(1, "Name: "+importData.Account.Name+" - "+currentData.HName, false)
		return true
	}
	if importData.Account.FirstName != "" && importData.Account.FirstName != currentData.HFirstName {
		logger(1, "FirstName: "+importData.Account.FirstName+" - "+currentData.HFirstName, false)
		return true
	}
	if importData.Account.LastName != "" && importData.Account.LastName != currentData.HLastName {
		logger(1, "LastName: "+importData.Account.LastName+" - "+currentData.HLastName, false)
		return true
	}
	if importData.Account.JobTitle != "" && importData.Account.JobTitle != currentData.HJobTitle {
		logger(1, "JobTitle: "+importData.Account.JobTitle+" - "+currentData.HJobTitle, false)
		return true
	}
	if importData.Account.Phone != "" && importData.Account.Phone != currentData.HPhone {
		logger(1, "Phone: "+importData.Account.Phone+" - "+currentData.HPhone, false)
		return true
	}
	if importData.Account.Email != "" && importData.Account.Email != currentData.HEmail {
		logger(1, "Email: "+importData.Account.Email+" - "+currentData.HEmail, false)
		return true
	}
	if importData.Account.Mobile != "" && importData.Account.Mobile != currentData.HMobile {
		logger(1, "Mobile: "+importData.Account.Mobile+" - "+currentData.HMobile, false)
		return true
	}
	if importData.Account.AbsenceMessage != "" && importData.Account.AbsenceMessage != currentData.HAvailStatusMsg {
		logger(1, "AbsenceMessage: "+importData.Account.AbsenceMessage+" - "+currentData.HAvailStatusMsg, false)
		return true
	}
	//-- If TimeZone mapping is empty then ignore as it defaults to a value
	if importData.Account.TimeZone != "" && importData.Account.TimeZone != currentData.HTimezone {
		logger(1, "TimeZone: "+importData.Account.TimeZone+" - "+currentData.HTimezone, false)
		return true
	}
	//-- If Language mapping is empty then ignore as it defaults to a value
	if importData.Account.Language != "" && importData.Account.Language != currentData.HLanguage {
		logger(1, "Language: "+importData.Account.Language+" - "+currentData.HLanguage, false)
		return true
	}
	//-- If DateTimeFormat mapping is empty then ignore as it defaults to a value
	if importData.Account.DateTimeFormat != "" && importData.Account.DateTimeFormat != currentData.HDateTimeFormat {
		logger(1, "DateTimeFormat: "+importData.Account.DateTimeFormat+" - "+currentData.HDateTimeFormat, false)
		return true
	}
	//-- If DateFormat mapping is empty then ignore as it defaults to a value
	if importData.Account.DateFormat != "" && importData.Account.DateFormat != currentData.HDateFormat {
		logger(1, "DateFormat: "+importData.Account.DateFormat+" - "+currentData.HDateFormat, false)
		return true
	}
	//-- If TimeFormat mapping is empty then ignore as it defaults to a value
	if importData.Account.TimeFormat != "" && importData.Account.TimeFormat != currentData.HTimeFormat {
		logger(1, "TimeFormat: "+importData.Account.TimeFormat+" - "+currentData.HTimeFormat, false)
		return true
	}
	//-- If CurrencySymbol mapping is empty then ignore as it defaults to a value
	if importData.Account.CurrencySymbol != "" && importData.Account.CurrencySymbol != currentData.HCurrencySymbol {
		logger(1, "CurrencySymbol: "+importData.Account.CurrencySymbol+" - "+currentData.HCurrencySymbol, false)
		return true
	}
	//-- If CountryCode mapping is empty then ignore as it defaults to a value
	if importData.Account.CountryCode != "" && importData.Account.CountryCode != currentData.HCountry {
		logger(1, "CountryCode: "+importData.Account.CountryCode+" - "+currentData.HCountry, false)
		return true
	}

	return false
}
func checkUserNeedsProfileUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {

	if importData.Profile.Manager != "" && importData.Profile.Manager != currentData.HManager {
		logger(1, "Manager: "+importData.Profile.Manager+" - "+currentData.HManager, false)
		return true
	}

	if importData.Profile.MiddleName != "" && importData.Profile.MiddleName != currentData.HMiddleName {
		logger(1, "MiddleName: "+importData.Profile.MiddleName+" - "+currentData.HMiddleName, false)
		return true
	}

	if importData.Profile.JobDescription != "" && importData.Profile.JobDescription != currentData.HSummary {
		logger(1, "JobDescription: "+importData.Profile.JobDescription+" - "+currentData.HSummary, false)
		return true
	}
	if importData.Profile.Qualifications != "" && importData.Profile.Qualifications != currentData.HQualifications {
		logger(1, "Qualifications: "+importData.Profile.Qualifications+" - "+currentData.HQualifications, false)
		return true
	}
	if importData.Profile.Interests != "" && importData.Profile.Interests != currentData.HInterests {
		logger(1, "Interests: "+importData.Profile.Interests+" - "+currentData.HInterests, false)
		return true
	}
	if importData.Profile.Expertise != "" && importData.Profile.Expertise != currentData.HSkills {
		logger(1, "Expertise: "+importData.Profile.Expertise+" - "+currentData.HSkills, false)
		return true
	}
	if importData.Profile.Gender != "" && importData.Profile.Gender != currentData.HGender {
		logger(1, "Gender: "+importData.Profile.Gender+" - "+currentData.HGender, false)
		return true
	}
	if importData.Profile.Dob != "" && importData.Profile.Dob != currentData.HDob {
		logger(1, "Dob: "+importData.Profile.Dob+" - "+currentData.HDob, false)
		return true
	}
	if importData.Profile.Nationality != "" && importData.Profile.Nationality != currentData.HNationality {
		logger(1, "Nationality: "+importData.Profile.Nationality+" - "+currentData.HNationality, false)
		return true
	}
	if importData.Profile.Religion != "" && importData.Profile.Religion != currentData.HReligion {
		logger(1, "Religion: "+importData.Profile.Religion+" - "+currentData.HReligion, false)
		return true
	}
	if importData.Profile.HomeTelephone != "" && importData.Profile.HomeTelephone != currentData.HHomeTelephoneNumber {
		logger(1, "HomeTelephone: "+importData.Profile.HomeTelephone+" - "+currentData.HHomeTelephoneNumber, false)
		return true
	}
	if importData.Profile.SocialNetworkA != "" && importData.Profile.SocialNetworkA != currentData.HSnA {
		logger(1, "SocialNetworkA: "+importData.Profile.SocialNetworkA+" - "+currentData.HSnA, false)
		return true
	}
	if importData.Profile.SocialNetworkB != "" && importData.Profile.SocialNetworkB != currentData.HSnB {
		logger(1, "SocialNetworkB: "+importData.Profile.SocialNetworkB+" - "+currentData.HSnB, false)
		return true
	}
	if importData.Profile.SocialNetworkC != "" && importData.Profile.SocialNetworkC != currentData.HSnC {
		logger(1, "SocialNetworkC: "+importData.Profile.SocialNetworkC+" - "+currentData.HSnC, false)
		return true
	}
	if importData.Profile.SocialNetworkD != "" && importData.Profile.SocialNetworkD != currentData.HSnD {
		logger(1, "SocialNetworkD: "+importData.Profile.SocialNetworkD+" - "+currentData.HSnD, false)
		return true
	}
	if importData.Profile.SocialNetworkG != "" && importData.Profile.SocialNetworkG != currentData.HSnE {
		logger(1, "SocialNetworkE: "+importData.Profile.SocialNetworkE+" - "+currentData.HSnE, false)
		return true
	}
	if importData.Profile.SocialNetworkG != "" && importData.Profile.SocialNetworkG != currentData.HSnF {
		logger(1, "SocialNetworkF: "+importData.Profile.SocialNetworkF+" - "+currentData.HSnF, false)
		return true
	}
	if importData.Profile.SocialNetworkG != "" && importData.Profile.SocialNetworkG != currentData.HSnG {
		logger(1, "SocialNetworkG: "+importData.Profile.SocialNetworkG+" - "+currentData.HSnG, false)
		return true
	}
	if importData.Profile.SocialNetworkH != "" && importData.Profile.SocialNetworkH != currentData.HSnH {
		logger(1, "SocialNetworkH: "+importData.Profile.SocialNetworkH+" - "+currentData.HSnH, false)
		return true
	}
	if importData.Profile.PersonalInterests != "" && importData.Profile.PersonalInterests != currentData.HPersonalInterests {
		logger(1, "PersonalInterests: "+importData.Profile.PersonalInterests+" - "+currentData.HPersonalInterests, false)
		return true
	}
	if importData.Profile.HomeAddress != "" && importData.Profile.HomeAddress != currentData.HHomeAddress {
		logger(1, "HomeAddress: "+importData.Profile.HomeAddress+" - "+currentData.HHomeAddress, false)
		return true
	}
	if importData.Profile.PersonalBlog != "" && importData.Profile.PersonalBlog != currentData.HBlog {
		logger(1, "PersonalBlog: "+importData.Profile.PersonalBlog+" - "+currentData.HBlog, false)
		return true
	}
	if importData.Profile.Attrib1 != "" && importData.Profile.Attrib1 != currentData.HAttrib1 {
		logger(1, "Attrib1: "+importData.Profile.Attrib1+" - "+currentData.HAttrib1, false)
		return true
	}
	if importData.Profile.Attrib2 != "" && importData.Profile.Attrib2 != currentData.HAttrib2 {
		logger(1, "Attrib2: "+importData.Profile.Attrib2+" - "+currentData.HAttrib2, false)
		return true
	}
	if importData.Profile.Attrib3 != "" && importData.Profile.Attrib3 != currentData.HAttrib3 {
		logger(1, "Attrib3: "+importData.Profile.Attrib3+" - "+currentData.HAttrib3, false)
		return true
	}
	if importData.Profile.Attrib4 != "" && importData.Profile.Attrib4 != currentData.HAttrib4 {
		logger(1, "Attrib4: "+importData.Profile.Attrib4+" - "+currentData.HAttrib4, false)
		return true
	}
	if importData.Profile.Attrib5 != "" && importData.Profile.Attrib5 != currentData.HAttrib5 {
		logger(1, "Attrib5: "+importData.Profile.Attrib5+" - "+currentData.HAttrib5, false)
		return true
	}
	if importData.Profile.Attrib6 != "" && importData.Profile.Attrib6 != currentData.HAttrib6 {
		logger(1, "Attrib6: "+importData.Profile.Attrib6+" - "+currentData.HAttrib6, false)
		return true
	}
	if importData.Profile.Attrib7 != "" && importData.Profile.Attrib7 != currentData.HAttrib7 {
		logger(1, "Attrib7: "+importData.Profile.Attrib7+" - "+currentData.HAttrib7, false)
		return true
	}
	if importData.Profile.Attrib8 != "" && importData.Profile.Attrib8 != currentData.HAttrib8 {
		logger(1, "Attrib8: "+importData.Profile.Attrib8+" - "+currentData.HAttrib8, false)
		return true
	}
	return false
}

//-- For Each Import Actions process the data
func processImportActions(l *map[string]interface{}) string {
	//-- Set User Account Attributes
	var data = new(userWorkingDataStruct)
	data.DB = *l
	//-- init map
	data.Account.UserID = getUserFieldValue(l, "UserID")
	data.Account.CheckID = getUserFieldValue(l, "UserID")

	switch googleImportConf.User.HornbillUserIDColumn {
	case "h_employee_id":
		data.Account.CheckID = getUserFieldValue(l, "EmployeeID")
	case "h_login_id":
		data.Account.CheckID = getUserFieldValue(l, "LoginID")
	case "h_email":
		data.Account.CheckID = getUserFieldValue(l, "Email")
	case "h_mobile":
		data.Account.CheckID = getUserFieldValue(l, "Mobile")
	}

	if data.Account.CheckID == "" {
		logger(3, "No Unique Identifier set for this record  "+fmt.Sprintf("%v", l), true)
		os.Exit(1)
	}
	logger(2, "Process Data for:  "+data.Account.CheckID+" ("+data.Account.UserID+")", false)
	//-- Loop Matches
	for _, action := range googleImportConf.Actions {
		switch action.Action {
		case "Regex":
			//-- Grab value from Google
			Outcome := processComplexField(l, action.Value)
			//-- Process Regex
			Outcome = processRegexOnString(action.Options.RegexValue, Outcome)
			//-- Store
			data.DB[action.Output] = Outcome
			logger(1, "Regex Output: "+Outcome, false)
		case "Replace":
			//-- Grab value from Google
			Outcome := processComplexField(l, action.Value)
			//-- Run Replace
			Outcome = strings.ReplaceAll(Outcome, action.Options.ReplaceOld, action.Options.ReplaceNew)
			//-- Store
			data.DB[action.Output] = Outcome
			logger(1, "Replace Output: "+Outcome, false)
		case "Trim":
			//-- Grab value from Google
			Outcome := processComplexField(l, action.Value)
			//-- Run Replace
			Outcome = strings.TrimSpace(Outcome)
			Outcome = strings.Replace(Outcome, "\n", "", -1)
			Outcome = strings.Replace(Outcome, "\r", "", -1)
			Outcome = strings.Replace(Outcome, "\r\n", "", -1)
			//-- Store
			data.DB[action.Output] = Outcome
			logger(1, "Trim Output: "+Outcome, false)
		default:
			logger(3, "Unknown Action: "+action.Action, false)
		}
	}
	//-- Store Result in map of userid
	var userID = strings.ToLower(data.Account.CheckID)
	HornbillCache.UsersWorking[userID] = data
	return userID
}

//-- For Each Google User Process Account And Mappings
func processUserParams(l *map[string]interface{}, userID string) {

	data := HornbillCache.UsersWorking[userID]
	data.Account.LoginID = getUserFieldValue(l, "LoginID")
	data.Account.EmployeeID = getUserFieldValue(l, "EmployeeID")
	data.Account.UserType = getUserFieldValue(l, "UserType")
	data.Account.Name = getUserFieldValue(l, "Name")
	data.Account.Password = getUserFieldValue(l, "Password")
	data.Account.FirstName = getUserFieldValue(l, "FirstName")
	data.Account.LastName = getUserFieldValue(l, "LastName")
	data.Account.JobTitle = getUserFieldValue(l, "JobTitle")
	data.Account.Site = getUserFieldValue(l, "Site")
	data.Account.Phone = getUserFieldValue(l, "Phone")
	data.Account.Email = getUserFieldValue(l, "Email")
	data.Account.Mobile = getUserFieldValue(l, "Mobile")
	data.Account.AbsenceMessage = getUserFieldValue(l, "AbsenceMessage")
	data.Account.TimeZone = getUserFieldValue(l, "TimeZone")
	data.Account.Language = getUserFieldValue(l, "Language")
	data.Account.DateTimeFormat = getUserFieldValue(l, "DateTimeFormat")
	data.Account.DateFormat = getUserFieldValue(l, "DateFormat")
	data.Account.TimeFormat = getUserFieldValue(l, "TimeFormat")
	data.Account.CurrencySymbol = getUserFieldValue(l, "CurrencySymbol")
	data.Account.CountryCode = getUserFieldValue(l, "CountryCode")

	data.Profile.MiddleName = getProfileFieldValue(l, "MiddleName")
	data.Profile.Manager = getProfileFieldValue(l, "Manager")
	data.Profile.JobDescription = getProfileFieldValue(l, "JobDescription")
	data.Profile.Qualifications = getProfileFieldValue(l, "Qualifications")
	data.Profile.Interests = getProfileFieldValue(l, "Interests")
	data.Profile.Expertise = getProfileFieldValue(l, "Expertise")
	data.Profile.Gender = getProfileFieldValue(l, "Gender")
	data.Profile.Dob = getProfileFieldValue(l, "Dob")
	data.Profile.Nationality = getProfileFieldValue(l, "Nationality")
	data.Profile.Religion = getProfileFieldValue(l, "Religion")
	data.Profile.HomeTelephone = getProfileFieldValue(l, "HomeTelephone")
	data.Profile.SocialNetworkA = getProfileFieldValue(l, "SocialNetworkA")
	data.Profile.SocialNetworkB = getProfileFieldValue(l, "SocialNetworkB")
	data.Profile.SocialNetworkC = getProfileFieldValue(l, "SocialNetworkC")
	data.Profile.SocialNetworkD = getProfileFieldValue(l, "SocialNetworkD")
	data.Profile.SocialNetworkE = getProfileFieldValue(l, "SocialNetworkE")
	data.Profile.SocialNetworkF = getProfileFieldValue(l, "SocialNetworkF")
	data.Profile.SocialNetworkG = getProfileFieldValue(l, "SocialNetworkG")
	data.Profile.SocialNetworkH = getProfileFieldValue(l, "SocialNetworkH")
	data.Profile.PersonalInterests = getProfileFieldValue(l, "PersonalInterests")
	data.Profile.HomeAddress = getProfileFieldValue(l, "HomeAddress")
	data.Profile.PersonalBlog = getProfileFieldValue(l, "PersonalBlog")
	data.Profile.Attrib1 = getProfileFieldValue(l, "Attrib1")
	data.Profile.Attrib2 = getProfileFieldValue(l, "Attrib2")
	data.Profile.Attrib3 = getProfileFieldValue(l, "Attrib3")
	data.Profile.Attrib4 = getProfileFieldValue(l, "Attrib4")
	data.Profile.Attrib5 = getProfileFieldValue(l, "Attrib5")
	data.Profile.Attrib6 = getProfileFieldValue(l, "Attrib6")
	data.Profile.Attrib7 = getProfileFieldValue(l, "Attrib7")
	data.Profile.Attrib8 = getProfileFieldValue(l, "Attrib8")
}
