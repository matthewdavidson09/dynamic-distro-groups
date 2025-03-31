package sync

import (
	"context"
	"fmt"
	"os"
	"strings"
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
		// 1. Sync to Active Directory
		adAdded, adRemoved, err := active_directory.SyncGroupByCategory(client, "all", "employees", users, dryRun)
		if err != nil {
			tools.Log.Errorf("Failed to sync All Employees group in AD: %v", err)
			return
		}

		// 2. Group identifiers
		groupCN := "list-all-employees"
		groupEmail := fmt.Sprintf("%s@%s", groupCN, os.Getenv("GROUP_EMAIL_DOMAIN"))
		groupName := "All Employees"

		// 3. Extract all emails and manager map
		var allEmails []string
		managerMap := make(map[string]bool)

		for _, user := range users {
			email := strings.ToLower(strings.TrimSpace(user.Email))
			if email == "" {
				continue
			}
			allEmails = append(allEmails, email)
			if len(user.DirectReports) > 0 {
				managerMap[email] = true
			}
		}

		// 4. Create Google Directory client
		svc, err := googleclient.NewDirectoryService(ctx)
		if err != nil {
			tools.Log.Errorf("Failed to create Google Directory client: %v", err)
			return
		}

		// 5. Build valid member list
		memberEmails := BuildMailboxAllowedList(svc, allEmails)

		// 6. Sync to Google Workspace (with roles)
		gAdded, gRemoved, err := SyncGoogleGroupWithRoles(ctx, svc, groupEmail, groupName, memberEmails, managerMap, dryRun)
		if err != nil {
			tools.Log.Errorf("Google group sync failed: %v", err)
		}

		// 7. Apply group settings to enforce managers-only posting
		if err := ApplyGoogleGroupSettings(ctx, groupEmail); err != nil {
			tools.Log.WithField("all", memberEmails).Errorf("Failed to apply Google group settings: %v", err)
		} else {
			tools.Log.WithField("all", memberEmails).Infof("Successfully applied Google group settings to %s", groupEmail)
		}

		// 8. Unified sync summary
		tools.LogSyncCombined(tools.SyncMetrics{
			GroupEmail:    groupEmail,
			TotalUsers:    len(memberEmails),
			ADAdded:       adAdded,
			ADRemoved:     adRemoved,
			GoogleAdded:   gAdded,
			GoogleRemoved: gRemoved,
		})
	})

	tools.Log.Infof("Finished All Employees sync in %s", time.Since(start))
}
