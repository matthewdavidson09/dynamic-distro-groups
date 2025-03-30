package sync

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/googleclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

func SyncAllEmployees(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) {
	ctx := context.Background()

	tools.Log.Info("Syncing All Employees distribution list...")
	tools.Log.Infof("Found %d eligible employees", len(users))

	// 1. Sync to AD
	added, removed, err := active_directory.SyncGroupByCategory(client, "all", "employees", users, dryRun)
	if err != nil {
		tools.Log.Errorf("Failed to sync All Employees group in AD: %v", err)
		return
	}
	tools.LogSyncSummary("all", "employees", len(users), added, removed)

	// 2. Prep Google Workspace values
	groupCN := "list-all-employees"
	groupEmail := fmt.Sprintf("%s@%s", groupCN, os.Getenv("GROUP_EMAIL_DOMAIN"))
	groupName := "All Employees"

	var memberEmails []string
	for _, user := range users {
		if user.Email != "" {
			memberEmails = append(memberEmails, strings.ToLower(user.Email))
		}
	}

	// 3. Google Directory Sync
	svc, err := googleclient.NewDirectoryService(ctx)
	if err != nil {
		tools.Log.Errorf("Failed to create Google Directory service: %v", err)
		return
	}

	if err := SyncGoogleGroup(ctx, svc, groupEmail, groupName, memberEmails, dryRun); err != nil {
		tools.Log.Errorf("Google group sync failed: %v", err)
	}
}
