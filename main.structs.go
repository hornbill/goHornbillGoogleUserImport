package main

import (
	"sync"
	"time"

	apiLib "github.com/hornbill/goApiLib"
)

//----- Constants -----
const (
	version          = "1.0.0"
	repo             = "hornbill/goHornbillGoogleUserImport"
	appName          = "goHornbillGoogleUserImport"
	applicationName  = "Google Workspace User Import Utility"
	maxGoogleResults = 200
)

var (
	//Default page size quen caching Hornbill records. Will be used if we can't get the maxPageSize from the instance
	maxHornbillResults = 1000

	//Map of Hornbill organization types
	orgTypes = map[string]int{
		"general":    0,
		"team":       1,
		"department": 2,
		"costcenter": 3,
		"division":   4,
		"company":    5,
	}

	//Config to load from JSON
	googleImportConf googleImportConfStruct

	// Global XMLMC instance for querying Google via iBridge
	gEspXmlmc *apiLib.XmlmcInstStruct

	Time struct {
		timeNow   string
		startTime time.Time
	}

	counters struct {
		errors         uint16
		updated        uint16
		profileUpdated uint16
		imageUpdated   uint16
		groupUpdated   uint16
		groupsRemoved  uint16
		rolesUpdated   uint16

		statusUpdated uint16

		created uint16

		traffic uint64
	}

	mutexCounters = &sync.Mutex{}
	bufferMutex   = &sync.Mutex{}

	localGoogleUsers []map[string]interface{}

	//Password profiles
	passwordProfile       passwordProfileStruct
	blacklistURLs         = [...]string{"https://files.hornbill.com/hornbillStatic/password_blacklists/SplashData.txt", "https://files.hornbill.com/hornbillStatic/password_blacklists/Imperva.txt"}
	defaultPasswordLength = 10

	HornbillUserStatusMap = map[string]string{
		"0": "active",
		"1": "suspended",
		"2": "archived",
	}

	// Flags List
	Flags struct {
		configLogPrefix  string
		configDryRun     bool
		configVersion    bool
		configInstanceID string
		configAPIKey     string
		configWorkers    int
		configAPITimeout int
		configDebug      bool
		configFileName   string
	}

	// HornbillCache Struct
	HornbillCache struct {
		Users             map[string]userAccountStruct
		Sites             map[string]siteStruct
		UserRoles         map[string][]string
		UserGroups        map[string][]string
		Groups            map[string]userGroupStruct
		GroupsID          map[string]userGroupStruct
		UsersWorking      map[string]*userWorkingDataStruct
		UsersWorkingIndex map[int]*userWorkingDataStruct
	}
)

type passwordProfileStruct struct {
	Length              int
	UseLower            bool
	ForceLower          int
	UseUpper            bool
	ForceUpper          int
	UseNumeric          bool
	ForceNumeric        int
	UseSpecial          bool
	ForceSpecial        int
	Blacklist           []string
	CheckMustNotContain bool
}

type xmlmcSettingResponse struct {
	Params struct {
		Option []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"option"`
	} `json:"params"`
	State        stateJSONStruct `json:"state"`
	MethodResult bool            `json:"@status"`
}

type userImportJobs struct {
	id            string
	create        bool
	update        bool
	updateHomeOrg bool
	updateProfile bool
	updateType    bool
	updateSite    bool
	updateImage   bool
	updateStatus  bool
}
type userWorkingDataStruct struct {
	Account        AccountMappingStruct
	Profile        ProfileMappingStruct
	Image          googleImageStruct
	DB             map[string]interface{}
	Jobs           userImportJobs
	Roles          []string
	Groups         []userGroupStruct
	GroupsToRemove []string
}
type userGroupStruct struct {
	ID                     string
	Name                   string
	Type                   int
	Membership             string
	TasksView              bool
	TasksAction            bool
	OnlyOneGroupAssignment bool
}

//----- Structures -----
type googleImportConfStruct struct {
	GoogleConf struct {
		KeysafeID   int    `json:"KeysafeID"`
		Customer    string `json:"Customer"`
		Domain      string `json:"Domain"`
		Query       string `json:"Query"`
		ShowDeleted bool   `json:"ShowDeleted"`
	} `json:"GoogleConf"`
	User struct {
		Operation            string               `json:"Operation"`
		AccountMapping       AccountMappingStruct `json:"AccountMapping"`
		HornbillUserIDColumn string               `json:"HornbillUserIDColumn"`
		Type                 struct {
			Action string `json:"Action"`
		} `json:"Type"`
		Status struct {
			Action string `json:"Action"`
			Value  string `json:"Value"`
		} `json:"Status"`
		Role struct {
			Action string   `json:"Action"`
			Roles  []string `json:"Roles"`
		} `json:"Role"`
		ProfileMapping ProfileMappingStruct `json:"ProfileMapping"`
		Image          struct {
			Action             string `json:"Action"`
			UploadType         string `json:"UploadType"`
			InsecureSkipVerify bool   `json:"InsecureSkipVerify"`
			ImageType          string `json:"ImageType"`
			ImageSize          string `json:"ImageSize"`
			URI                string `json:"URI"`
		} `json:"Image"`
		Site struct {
			Action string `json:"Action"`
			Value  string `json:"Value"`
		} `json:"Site"`
		Org []struct {
			Action  string `json:"Action"`
			Value   string `json:"Value"`
			Options struct {
				Type                   string `json:"Type"`
				Membership             string `json:"Membership"`
				TasksView              bool   `json:"TasksView"`
				TasksAction            bool   `json:"TasksAction"`
				OnlyOneGroupAssignment bool   `json:"OnlyOneGroupAssignment"`
				SetAsHomeOrganisation  bool   `json:"SetAsHomeOrganisation"`
			} `json:"Options"`
		} `json:"Org"`
	} `json:"User"`
	Advanced struct {
		LogLevel     int `json:"LogLevel"`
		LogRetention int `json:"LogRetention"`
	} `json:"Advanced"`
	Actions []struct {
		Action  string `json:"Action"`
		Value   string `json:"Value"`
		Output  string `json:"Output"`
		Options struct {
			RegexValue string `json:"RegexValue"`
			ReplaceOld string `json:"ReplaceOld"`
			ReplaceNew string `json:"ReplaceNew"`
		} `json:"Options"`
	} `json:"Actions"`
}

// AccountMappingStruct Used
type AccountMappingStruct struct {
	UserID         string `json:"UserID"`
	LoginID        string `json:"LoginId"`
	CheckID        string
	EmployeeID     string `json:"EmployeeId"`
	UserType       string `json:"UserType"`
	Name           string `json:"Name"`
	Password       string `json:"Password"`
	FirstName      string `json:"FirstName"`
	LastName       string `json:"LastName"`
	JobTitle       string `json:"JobTitle"`
	Site           string `json:"Site"`
	Phone          string `json:"Phone"`
	Email          string `json:"Email"`
	Mobile         string `json:"Mobile"`
	AbsenceMessage string `json:"AbsenceMessage"`
	TimeZone       string `json:"TimeZone"`
	Language       string `json:"Language"`
	DateTimeFormat string `json:"DateTimeFormat"`
	DateFormat     string `json:"DateFormat"`
	TimeFormat     string `json:"TimeFormat"`
	CurrencySymbol string `json:"CurrencySymbol"`
	CountryCode    string `json:"CountryCode"`
	HomeOrg        string `json:"HomeOrg"`
}

// ProfileMappingStruct Used
type ProfileMappingStruct struct {
	MiddleName        string `json:"middleName"`
	JobDescription    string `json:"jobDescription"`
	Manager           string `json:"manager"`
	Qualifications    string `json:"qualifications"`
	Interests         string `json:"interests"`
	Expertise         string `json:"expertise"`
	Gender            string `json:"gender"`
	Dob               string `json:"dob"`
	Nationality       string `json:"nationality"`
	Religion          string `json:"religion"`
	HomeTelephone     string `json:"homeTelephone"`
	SocialNetworkA    string `json:"socialNetworkA"`
	SocialNetworkB    string `json:"socialNetworkB"`
	SocialNetworkC    string `json:"socialNetworkC"`
	SocialNetworkD    string `json:"socialNetworkD"`
	SocialNetworkE    string `json:"socialNetworkE"`
	SocialNetworkF    string `json:"socialNetworkF"`
	SocialNetworkG    string `json:"socialNetworkG"`
	SocialNetworkH    string `json:"socialNetworkH"`
	PersonalInterests string `json:"personalInterests"`
	HomeAddress       string `json:"homeAddress"`
	PersonalBlog      string `json:"personalBlog"`
	Attrib1           string `json:"Attrib1"`
	Attrib2           string `json:"Attrib2"`
	Attrib3           string `json:"Attrib3"`
	Attrib4           string `json:"Attrib4"`
	Attrib5           string `json:"Attrib5"`
	Attrib6           string `json:"Attrib6"`
	Attrib7           string `json:"Attrib7"`
	Attrib8           string `json:"Attrib8"`
}

type xmlmcSiteListResponse struct {
	Params struct {
		RowData struct {
			Row []siteStruct `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}
type xmlmcUserListResponse struct {
	Params struct {
		RowData struct {
			Row []userAccountStruct `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}
type siteStruct struct {
	HID       string `json:"h_id"`
	HSiteName string `json:"h_site_name"`
}
type roleStruct struct {
	HUserID string `json:"h_user_id"`
	HRole   string `json:"h_role"`
}
type groupStruct struct {
	HID   string `json:"h_id"`
	HName string `json:"h_name"`
	HType string `json:"h_type"`
}
type userAccountStruct struct {
	HUserID              string `json:"h_user_id"`
	HLoginID             string `json:"h_login_id"`
	HEmployeeID          string `json:"h_employee_id"`
	HName                string `json:"h_name"`
	HFirstName           string `json:"h_first_name"`
	HMiddleName          string `json:"h_middle_name"`
	HLastName            string `json:"h_last_name"`
	HPhone               string `json:"h_phone"`
	HEmail               string `json:"h_email"`
	HMobile              string `json:"h_mobile"`
	HJobTitle            string `json:"h_job_title"`
	HLoginCreds          string `json:"h_login_creds"`
	HClass               string `json:"h_class"`
	HAvailStatus         string `json:"h_avail_status"`
	HAvailStatusMsg      string `json:"h_avail_status_msg"`
	HTimezone            string `json:"h_timezone"`
	HCountry             string `json:"h_country"`
	HLanguage            string `json:"h_language"`
	HDateTimeFormat      string `json:"h_date_time_format"`
	HDateFormat          string `json:"h_date_format"`
	HTimeFormat          string `json:"h_time_format"`
	HCurrencySymbol      string `json:"h_currency_symbol"`
	HLastLogon           string `json:"h_last_logon"`
	HSnA                 string `json:"h_sn_a"`
	HSnB                 string `json:"h_sn_b"`
	HSnC                 string `json:"h_sn_c"`
	HSnD                 string `json:"h_sn_d"`
	HSnE                 string `json:"h_sn_e"`
	HSnF                 string `json:"h_sn_f"`
	HSnG                 string `json:"h_sn_g"`
	HSnH                 string `json:"h_sn_h"`
	HIconRef             string `json:"h_icon_ref"`
	HIconChecksum        string `json:"h_icon_checksum"`
	HDob                 string `json:"h_dob"`
	HAccountStatus       string `json:"h_account_status"`
	HFailedAttempts      string `json:"h_failed_attempts"`
	HIdxRef              string `json:"h_idx_ref"`
	HSite                string `json:"h_site"`
	HManager             string `json:"h_manager"`
	HSummary             string `json:"h_summary"`
	HInterests           string `json:"h_interests"`
	HQualifications      string `json:"h_qualifications"`
	HPersonalInterests   string `json:"h_personal_interests"`
	HSkills              string `json:"h_skills"`
	HGender              string `json:"h_gender"`
	HNationality         string `json:"h_nationality"`
	HReligion            string `json:"h_religion"`
	HHomeTelephoneNumber string `json:"h_home_telephone_number"`
	HHomeAddress         string `json:"h_home_address"`
	HBlog                string `json:"h_blog"`
	HAttrib1             string `json:"h_attrib1"`
	HAttrib2             string `json:"h_attrib2"`
	HAttrib3             string `json:"h_attrib3"`
	HAttrib4             string `json:"h_attrib4"`
	HAttrib5             string `json:"h_attrib5"`
	HAttrib6             string `json:"h_attrib6"`
	HAttrib7             string `json:"h_attrib7"`
	HAttrib8             string `json:"h_attrib8"`
	HHomeOrg             string `json:"h_home_organization"`
}
type xmlmcUserRolesListResponse struct {
	Params struct {
		RowData struct {
			Row []roleStruct `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}
type xmlmcUserGroupListResponse struct {
	Params struct {
		RowData struct {
			Row []struct {
				HUserID  string `json:"h_user_id"`
				HGroupID string `json:"h_group_id"`
			} `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}

type xmlmcGroupListResponse struct {
	Params struct {
		RowData struct {
			Row []groupStruct `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}
type xmlmcCountResponse struct {
	Params struct {
		RowData struct {
			Row []struct {
				Count string `json:"count"`
			} `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}
type stateJSONStruct struct {
	Code      string `json:"code"`
	Service   string `json:"service"`
	Operation string `json:"operation"`
	Error     string `json:"error"`
}
type xmlmcResponse struct {
	MethodResult string          `json:"status"`
	State        stateJSONStruct `json:"state"`
}

type xmlmcIBridgeResponse struct {
	MethodResult           string         `xml:"status,attr"`
	IBridgeResponsePayload string         `xml:"params>responsePayload"`
	IBridgeResponseError   string         `xml:"params>error"`
	State                  stateXMLStruct `xml:"state"`
}

type stateXMLStruct struct {
	Code  string `xml:"code"`
	Error string `xml:"error"`
}

type usersPayloadStruct struct {
	Customer    string `json:"customer"`
	Domain      string `json:"domain"`
	MaxResults  int    `json:"maxResults"`
	PageToken   string `json:"pageToken"`
	Query       string `json:"query"`
	ShowDeleted bool   `json:"showDeleted"`
}

type usersImagePayloadStruct struct {
	UserKey string `json:"userKey"`
}

type googleResponseStruct struct {
	Params struct {
		Data struct {
			Etag          string                           `json:"etag"`
			Groups        []googleGroupDetailsStruct       `json:"groups"`
			ImgHeight     int                              `json:"height"`
			ImgMimeType   string                           `json:"mimeType"`
			ImgPhotoData  string                           `json:"photoData"`
			ImgWidth      int                              `json:"width"`
			Members       []googleGroupMemberDetailsStruct `json:"members"`
			Users         []map[string]interface{}         `json:"users"`
			Kind          string                           `json:"kind"`
			NextPageToken string                           `json:"nextPageToken"`
		} `json:"data"`
		Error   string `json:"error"`
		Status  int    `json:"status"`
		Success bool   `json:"success"`
		URL     string `json:"url"`
	} `json:"params"`
}

type googleImageStruct struct {
	Height    int
	MimeType  string
	PhotoData string
	Width     int
}

type googleGroupDetailsStruct struct {
	AdminCreated       bool   `json:"adminCreated"`
	Description        string `json:"description"`
	DirectMembersCount string `json:"directMembersCount"`
	Email              string `json:"email"`
	Etag               string `json:"etag"`
	ID                 string `json:"id"`
	Kind               string `json:"kind"`
	Name               string `json:"name"`
	Members            map[string]googleGroupMemberDetailsStruct
}

type googleGroupMemberDetailsStruct struct {
	Email string `json:"email"`
	Etag  string `json:"etag"`
	ID    string `json:"id"`
	Role  string `json:"role"`
	Type  string `json:"type"`
}
