package adsync

import (
	"strings"
	"time"

	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

func SyncStates(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) {
	states := active_directory.GetUniqueStates(users)

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

		added, removed, err := active_directory.SyncGroupByCategory(client, "state", state, stateUsers, dryRun)
		tools.LogSyncSummary("state", state, len(stateUsers), added, removed)

		if err != nil {
			tools.Log.WithField("state", state).Errorf("Sync error: %v", err)
		}
	})

	tools.Log.Infof("Finished syncing states in %s", time.Since(start))
}
