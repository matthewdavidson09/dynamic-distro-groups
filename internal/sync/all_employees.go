package sync

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/googleclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

func SyncAllEmployees(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) {
	ctx := context.Background()
	start := time.Now()

	tools.Log.Info("Syncing All Employees distribution list...")

	tools.RunWithWorkers([]string{"employees"}, 1, func(_ string) {
		// 1. Sync to AD
		adAdded, adRemoved, err := active_directory.SyncGroupByCategory(client, "all", "employees", users, dryRun)
		if err != nil {
			tools.Log.Errorf("Failed to sync All Employees group in AD: %v", err)
			return
		}

		// 2. Prepare Google Group info
		groupCN := "list-all-employees"
		groupEmail := fmt.Sprintf("%s@%s", groupCN, os.Getenv("GROUP_EMAIL_DOMAIN"))
		groupName := "All Employees"

		// 3. Prepare all emails
		var allEmails []string
		for _, user := range users {
			if user.Email != "" {
				allEmails = append(allEmails, user.Email)
			}
		}

		// 4. Create Google client
		svc, err := googleclient.NewDirectoryService(ctx)
		if err != nil {
			tools.Log.Errorf("Failed to create Google Directory service: %v", err)
			return
		}

		// 5. Perform mailbox checks in parallel
		memberEmails := BuildMailboxAllowedList(svc, allEmails)

		// 6. Sync to Google
		gAdded, gRemoved, err := SyncGoogleGroup(ctx, svc, groupEmail, groupName, memberEmails, dryRun)
		if err != nil {
			tools.Log.Errorf("Google group sync failed: %v", err)
		}

		// 7. Log combined result
		tools.LogSyncCombined(tools.SyncMetrics{
			GroupEmail:    groupEmail,
			TotalUsers:    len(users),
			ADAdded:       adAdded,
			ADRemoved:     adRemoved,
			GoogleAdded:   gAdded,
			GoogleRemoved: gRemoved,
		})
	})

	tools.Log.Infof("Finished All Employees sync in %s", time.Since(start))
}
