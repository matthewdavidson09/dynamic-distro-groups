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

func SyncStates(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) {
	states := active_directory.GetUniqueStates(users)
	ctx := context.Background()

	start := time.Now()
	tools.Log.Infof("Syncing %d state-based groups...", len(states))

	// Index for quick manager detection
	adLookup := make(map[string]active_directory.ADUser)
	for _, u := range users {
		adLookup[strings.ToLower(strings.TrimSpace(u.DN))] = u
	}

	tools.RunWithWorkers(states, 5, func(state string) {
		var stateUsers []active_directory.ADUser
		for _, user := range users {
			if strings.EqualFold(strings.TrimSpace(user.State), state) {
				stateUsers = append(stateUsers, user)
			}
		}

		if len(stateUsers) == 0 {
			tools.Log.WithField("state", state).Warn("No users found, skipping.")
			return
		}

		// 1. Sync to Active Directory
		adAdded, adRemoved, err := active_directory.SyncGroupByCategory(client, "state", state, stateUsers, dryRun)
		if err != nil {
			tools.Log.WithField("state", state).Errorf("AD sync error: %v", err)
			return
		}

		// 2. Build Google Group info
		groupEmail := fmt.Sprintf("list-state-%s@%s", tools.Slugify(state), os.Getenv("GROUP_EMAIL_DOMAIN"))
		groupName := fmt.Sprintf("State: %s", state)

		// 3. Determine members and manager role eligibility
		memberEmails := []string{}
		managerEmails := map[string]bool{}
		for _, user := range stateUsers {
			email := strings.ToLower(strings.TrimSpace(user.Email))
			if email == "" {
				continue
			}
			memberEmails = append(memberEmails, email)

			// If this user manages others, mark them as a manager
			if len(user.DirectReports) > 0 {
				managerEmails[email] = true
			}
		}

		// 4. Sync to Google Workspace
		svc, err := googleclient.NewDirectoryService(ctx)
		if err != nil {
			tools.Log.WithField("state", state).Errorf("Failed to create Google Directory client: %v", err)
			return
		}

		gAdded, gRemoved, err := SyncGoogleGroupWithRoles(ctx, svc, groupEmail, groupName, memberEmails, managerEmails, dryRun)
		if err != nil {
			tools.Log.WithField("state", state).Errorf("Google group sync error: %v", err)
		}

		// 5. Apply group settings to enforce managers-only posting
		if err := ApplyGoogleGroupSettings(ctx, groupEmail); err != nil {
			tools.Log.WithField("state", state).Errorf("Failed to apply Google group settings: %v", err)
		} else {
			tools.Log.WithField("state", state).Infof("Successfully applied Google group settings to %s", groupEmail)
		}

		// 6. Unified Logging
		tools.LogSyncCombined(tools.SyncMetrics{
			GroupEmail:    groupEmail,
			TotalUsers:    len(stateUsers),
			ADAdded:       adAdded,
			ADRemoved:     adRemoved,
			GoogleAdded:   gAdded,
			GoogleRemoved: gRemoved,
		})
	})

	tools.Log.Infof("Finished syncing states in %s", time.Since(start))
}
