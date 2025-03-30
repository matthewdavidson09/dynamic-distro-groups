package adsync

import (
	"strings"
	"time"

	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

// RunAllGroupSyncs orchestrates department, state, and "all employees" syncs.
func RunAllGroupSyncs(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) error {

	departments := active_directory.GetUniqueDepartments(users)
	states := active_directory.GetUniqueStates(users)

	// Sync by Department
	start := time.Now()
	tools.Log.Infof("Syncing %d department-based groups...", len(departments))
	tools.RunWithWorkers(departments, 5, func(dept string) {
		var deptUsers []active_directory.ADUser
		for _, user := range users {
			if strings.EqualFold(strings.TrimSpace(user.Department), dept) {
				deptUsers = append(deptUsers, user)
			}
		}
		if len(deptUsers) == 0 {
			tools.Log.WithField("dept", dept).Warn("No users found, skipping.")
			return
		}

		added, removed, err := active_directory.SyncGroupByCategory(client, "dept", dept, deptUsers, dryRun)
		tools.LogSyncSummary("dept", dept, len(deptUsers), added, removed)
		if err != nil {
			tools.Log.WithField("dept", dept).Errorf("Sync error: %v", err)
		}
	})
	tools.Log.Infof("Finished syncing departments in %s", time.Since(start))

	// Sync by State
	startStates := time.Now()
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
	tools.Log.Infof("Finished syncing states in %s", time.Since(startStates))

	// Sync All Employees Group
	tools.Log.Info("Syncing All Employees distribution list...")
	tools.Log.Infof("Found %d eligible employees", len(users))
	added, removed, err := active_directory.SyncGroupByCategory(client, "all", "employees", users, dryRun)
	if err != nil {
		return err
	}
	tools.LogSyncSummary("all", "employees", len(users), added, removed)

	return nil
}
