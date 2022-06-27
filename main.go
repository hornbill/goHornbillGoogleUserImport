package main

//----- Packages -----
import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	apiLib "github.com/hornbill/goApiLib"
)

var (
	onceLog   sync.Once
	loggerAPI *apiLib.XmlmcInstStruct
	mutexLog  = &sync.Mutex{}
	f         *os.File
)

// Main
func main() {
	//-- Start Time for Durration
	Time.startTime = time.Now()
	//-- Start Time for Log File
	Time.timeNow = time.Now().Format("20060102150405.000000")

	//-- Process Flags
	procFlags()

	//-- Check for latest, update if necessary
	doSelfUpdate()

	//-- Load Configuration File Into Struct
	googleImportConf = loadConfig()

	setTemplateFilters()
	templateFault := checkTemplate()
	if templateFault {
		logger(4, " [Template] Issues were found with the template.", true)
		return
	}

	loggerAPI = apiLib.NewXmlmcInstance(Flags.configInstanceID)
	loggerAPI.SetAPIKey(Flags.configAPIKey)
	loggerAPI.SetTimeout(Flags.configAPITimeout)
	loggerAPI.SetJSONResponse(true)

	maxResultsAllowed, err := strconv.Atoi(sysOptionGet("api.xmlmc.queryExec.maxResultsAllowed"))
	if err == nil {
		maxHornbillResults = maxResultsAllowed
	}

	//-- Clear Old Log Files
	runLogRetentionCheck()

	//-- Get Password Profile
	getPasswordProfile()

	googleImportConf.User.HornbillUserIDColumn = strings.ToLower(googleImportConf.User.HornbillUserIDColumn)

	//-- Query Google
	getGoogleUsers()

	//-- Process Google User Data First
	//-- So we only store data about users we have
	processGoogleUsers()

	//-- Fetch Users from Hornbill
	loadUsers()

	//-- Load User Roles
	loadUsersRoles()

	//-- Fetch Sites
	loadSites()

	//-- Fetch Groups
	loadGroups()

	//-- Fetch User Groups
	loadUserGroups()

	//-- Create List of Actions that need to happen
	processData()

	//-- Run Actions
	finaliseData()

	//-- End Ouput
	outputEnd()
}

//-- Process Input Flags
func procFlags() {
	//-- Grab Flags
	flag.StringVar(&Flags.configInstanceID, "instance", "", "ID of the Hornbill instance")
	flag.StringVar(&Flags.configAPIKey, "apikey", "", "API Key to authenticate the Hornbill requests")
	flag.StringVar(&Flags.configFileName, "file", "conf.json", "Name of Configuration File To Load")
	flag.StringVar(&Flags.configLogPrefix, "logprefix", "", "Add prefix to the logfile")
	flag.BoolVar(&Flags.configDryRun, "dryrun", false, "Allow the Import to run without Creating or Updating users")
	flag.BoolVar(&Flags.configVersion, "version", false, "Output Version")
	flag.IntVar(&Flags.configWorkers, "workers", 1, "Number of Worker threads to use")
	flag.IntVar(&Flags.configAPITimeout, "apitimeout", 60, "Number of Seconds to Timeout an API Connection")
	flag.BoolVar(&Flags.configDebug, "debug", false, "Debug level logging")

	//-- Parse Flags
	flag.Parse()

	//-- Used for building release packages
	if Flags.configVersion {
		fmt.Printf("%v \n", version)
		os.Exit(0)
	}

	logger(0, "\n=== "+applicationName+" v"+fmt.Sprintf("%v", version)+" ===\n", true)

	//-- Output config
	logger(1, "Flag - instance "+Flags.configInstanceID, true)
	logger(1, "Flag - apikey "+Flags.configAPIKey, true)
	logger(1, "Flag - file "+Flags.configFileName, true)
	logger(1, "Flag - logprefix "+Flags.configLogPrefix, true)
	logger(1, "Flag - workers "+fmt.Sprintf("%v", Flags.configWorkers), false)
	logger(1, "Flag - apitimeout "+fmt.Sprintf("%v", Flags.configAPITimeout), true)
	logger(1, "Flag - dryrun "+fmt.Sprintf("%v", Flags.configDryRun), true)
	logger(1, "Flag - debug "+fmt.Sprintf("%v", Flags.configDebug)+"\n", true)

}

//-- Generate Output
func outputEnd() {
	logger(0, "\n=== Import Process Summary ===", true)
	//-- End output
	if counters.errors > 0 {
		logger(4, "One or more errors encountered, please check the log file", true)
	}
	logger(2, "Error Count: "+fmt.Sprintf("%d", counters.errors), true)
	logger(2, "Accounts Processed: "+fmt.Sprintf("%d", len(HornbillCache.UsersWorking)), true)
	logger(2, "Created: "+fmt.Sprintf("%d", counters.created), true)
	logger(2, "Updated: "+fmt.Sprintf("%d", counters.updated), true)
	logger(2, "Status Updates: "+fmt.Sprintf("%d", counters.statusUpdated), true)
	logger(2, "Profiles Updated: "+fmt.Sprintf("%d", counters.profileUpdated), true)
	logger(2, "Images Updated: "+fmt.Sprintf("%d", counters.imageUpdated), true)
	logger(2, "Groups Updated: "+fmt.Sprintf("%d", counters.groupUpdated), true)
	logger(2, "Groups Removed: "+fmt.Sprintf("%d", counters.groupsRemoved), true)
	logger(2, "Roles Added: "+fmt.Sprintf("%d", counters.rolesUpdated), true)
	logger(2, "Time Taken: "+time.Since(Time.startTime).Round(time.Second).String(), true)

	mutexCounters.Lock()
	counters.traffic += loggerAPI.GetCount()
	counters.traffic += hornbillImport.GetCount()
	counters.traffic += gEspXmlmc.GetCount()
	mutexCounters.Unlock()
	logger(2, "Total Traffic: "+fmt.Sprintf("%d", counters.traffic), true)

	logger(0, "\n=== "+applicationName+" Import Complete ===\n", true)
}

func loadConfig() googleImportConfStruct {
	//-- Check Config File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + Flags.configFileName
	logger(1, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}

	//-- Load config file
	file, fileError := os.Open(configurationFilePath)
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
		os.Exit(102)
	}
	eConf := googleImportConfStruct{}

	//-- Decode JSON from config file
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&eConf)
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+fmt.Sprintf("%v", err), true)
		os.Exit(102)
	}
	return eConf
}

// CounterInc Generic Counter Increment
func CounterInc(counter int) {
	mutexCounters.Lock()
	switch counter {
	case 1:
		counters.created++
	case 2:
		counters.updated++
	case 3:
		counters.profileUpdated++
	case 4:
		counters.imageUpdated++
	case 5:
		counters.groupUpdated++
	case 6:
		counters.rolesUpdated++
	case 7:
		counters.errors++
	case 8:
		counters.groupsRemoved++
	case 9:
		counters.statusUpdated++
	}
	mutexCounters.Unlock()
}

func doSelfUpdate() {
	v := semver.MustParse(version)
	latest, found, err := selfupdate.DetectLatest(repo)
	if err != nil {
		logger(5, "Error occurred while detecting version: "+err.Error(), true)
		return
	}
	if !found {
		logger(5, "Could not find Github repo: "+repo, true)
		return
	}

	latestMajorVersion := strings.Split(fmt.Sprintf("%v", latest.Version), ".")[0]
	latestMinorVersion := strings.Split(fmt.Sprintf("%v", latest.Version), ".")[1]
	latestPatchVersion := strings.Split(fmt.Sprintf("%v", latest.Version), ".")[2]

	currentMajorVersion := strings.Split(version, ".")[0]
	currentMinorVersion := strings.Split(version, ".")[1]
	currentPatchVersion := strings.Split(version, ".")[2]

	//Useful in dev, customers should never see current version > latest release version
	if currentMajorVersion > latestMajorVersion {
		logger(3, "Current version "+version+" (major) is greater than the latest release version on Github "+fmt.Sprintf("%v", latest.Version), true)
		return
	} else {
		if currentMinorVersion > latestMinorVersion {
			logger(3, "Current version "+version+" (minor) is greater than the latest release version on Github "+fmt.Sprintf("%v", latest.Version), true)
			return
		} else if currentPatchVersion > latestPatchVersion {
			logger(3, "Current version "+version+" (patch) is greater than the latest release version on Github "+fmt.Sprintf("%v", latest.Version), true)
			return
		}
	}
	if latestMajorVersion > currentMajorVersion {
		msg := "v" + version + " is not latest, you should upgrade to " + fmt.Sprintf("%v", latest.Version) + " by downloading the latest package from: https://github.com/" + repo + "/releases/latest"
		logger(5, msg, true)
		return
	}

	_, err = selfupdate.UpdateSelf(v, repo)
	if err != nil {
		logger(5, "Binary update failed: "+err.Error(), true)
		return
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		logger(3, "Current binary is the latest version: "+version, true)
	} else {
		logger(3, "Successfully updated to version: "+fmt.Sprintf("%v", latest.Version), true)
		logger(3, "Release notes:\n"+latest.ReleaseNotes, true)
	}
}
