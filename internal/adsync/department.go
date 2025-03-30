package adsync

import (
	"strings"
	"time"

	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

func SyncDepartments(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) {
	departments := active_directory.GetUniqueDepartments(users)

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
}
