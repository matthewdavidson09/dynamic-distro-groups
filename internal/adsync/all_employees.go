package adsync

import (
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

func SyncAllEmployees(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) {
	tools.Log.Info("Syncing All Employees distribution list...")
	tools.Log.Infof("Found %d eligible employees", len(users))

	added, removed, err := active_directory.SyncGroupByCategory(client, "all", "employees", users, dryRun)
	if err != nil {
		tools.Log.Errorf("Failed to sync All Employees group: %v", err)
		return
	}

	tools.LogSyncSummary("all", "employees", len(users), added, removed)
}
