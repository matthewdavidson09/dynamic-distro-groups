package sync

import (
	"os"
	"strings"

	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

func RunAllGroupSyncs(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) error {
	targets := strings.Split(strings.ToLower(os.Getenv("SYNC_TARGETS")), ",")

	shouldRun := func(name string) bool {
		for _, t := range targets {
			if t == name || t == "all" {
				return true
			}
		}
		return false
	}

	if shouldRun("departments") {
		tools.Log.Info("Running department group sync...")
		SyncDepartments(client, users, dryRun)
	}
	if shouldRun("states") {
		tools.Log.Info("Running state group sync...")
		SyncStates(client, users, dryRun)
	}
	if shouldRun("managers") {
		tools.Log.Info("Running manager group sync...")
		SyncManagers(client, users, dryRun)
	}
	if shouldRun("all") || shouldRun("all-employees") {
		tools.Log.Info("Running all employees group sync...")
		SyncAllEmployees(client, users, dryRun)
	}

	return nil
}
