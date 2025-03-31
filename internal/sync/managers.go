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

func SyncManagers(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) {
	managerMap := active_directory.GroupUsersByManager(users)
	ctx := context.Background()

	tools.Log.Infof("Syncing %d manager-based distribution lists...", len(managerMap))

	tools.RunWithWorkers(tools.MapKeys(managerMap), 5, func(managerDN string) {
		// Find the manager user
		var manager *active_directory.ADUser
		for _, u := range users {
			if active_directory.NormalizeDN(u.DN) == active_directory.NormalizeDN(managerDN) {
				manager = &u
				break
			}
		}
		if manager == nil || manager.Email == "" {
			tools.Log.WithField("manager", managerDN).Warn("Manager not found or has no email.")
			return
		}

		groupEmail := fmt.Sprintf("list-reports-%s@%s", tools.Slugify(manager.SAMAccountName), os.Getenv("GROUP_EMAIL_DOMAIN"))
		groupName := fmt.Sprintf("Manager: %s", manager.DisplayName)

		// Include manager + direct reports
		members := append(managerMap[managerDN], *manager)

		// 1. Sync AD
		adAdded, adRemoved, err := active_directory.SyncGroupByCategory(client, "manager", manager.SAMAccountName, members, dryRun)
		if err != nil {
			tools.Log.WithField("manager", manager.SAMAccountName).Errorf("AD sync error: %v", err)
			return
		}

		// 2. Build Google Group
		var emails []string
		for _, u := range members {
			if u.Email != "" {
				emails = append(emails, strings.ToLower(u.Email))
			}
		}

		svc, err := googleclient.NewDirectoryService(ctx)
		if err != nil {
			tools.Log.WithField("manager", manager.SAMAccountName).Errorf("Google client error: %v", err)
			return
		}

		gAdded, gRemoved, err := SyncGoogleGroup(ctx, svc, groupEmail, groupName, emails, dryRun)
		if err != nil {
			tools.Log.WithField("manager", manager.SAMAccountName).Errorf("Google sync error: %v", err)
		}

		// Unified logging
		tools.LogSyncCombined(tools.SyncMetrics{
			GroupEmail:    groupEmail,
			TotalUsers:    len(members),
			ADAdded:       adAdded,
			ADRemoved:     adRemoved,
			GoogleAdded:   gAdded,
			GoogleRemoved: gRemoved,
		})
	})

	tools.Log.Info("Finished syncing manager-based distribution lists.")
}
