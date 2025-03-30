package adsync

import (
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
)

func RunAllGroupSyncs(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) error {
	SyncDepartments(client, users, dryRun)
	SyncStates(client, users, dryRun)
	SyncAllEmployees(client, users, dryRun)
	return nil
}
