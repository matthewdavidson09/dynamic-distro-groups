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

		// 1. Sync AD group
		added, removed, err := active_directory.SyncGroupByCategory(client, "state", state, stateUsers, dryRun)
		tools.LogSyncSummary("state", state, len(stateUsers), added, removed)

		if err != nil {
			tools.Log.WithField("state", state).Errorf("AD sync error: %v", err)
			return
		}

		// 2. Prepare Google group details
		groupEmail := fmt.Sprintf("list-state-%s@%s", tools.Slugify(state), os.Getenv("GROUP_EMAIL_DOMAIN"))
		groupName := fmt.Sprintf("State: %s", state)

		var memberEmails []string
		for _, user := range stateUsers {
			if user.Email != "" {
				memberEmails = append(memberEmails, strings.ToLower(user.Email))
			}
		}

		// 3. Sync to Google
		svc, err := googleclient.NewDirectoryService(ctx)
		if err != nil {
			tools.Log.WithField("state", state).Errorf("Failed to create Google Directory client: %v", err)
			return
		}

		if err := SyncGoogleGroup(ctx, svc, groupEmail, groupName, memberEmails, dryRun); err != nil {
			tools.Log.WithField("state", state).Errorf("Google group sync error: %v", err)
		}
	})

	tools.Log.Infof("Finished syncing states in %s", time.Since(start))
}
