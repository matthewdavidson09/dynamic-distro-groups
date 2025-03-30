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

func SyncDepartments(client *ldapclient.LDAPClient, users []active_directory.ADUser, dryRun bool) {
	departments := active_directory.GetUniqueDepartments(users)
	ctx := context.Background()

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

		// 1. Sync to Active Directory
		added, removed, err := active_directory.SyncGroupByCategory(client, "dept", dept, deptUsers, dryRun)
		tools.LogSyncSummary("dept", dept, len(deptUsers), added, removed)
		if err != nil {
			tools.Log.WithField("dept", dept).Errorf("AD sync error: %v", err)
			return
		}

		// 2. Build Google Group info
		groupEmail := fmt.Sprintf("list-dept-%s@%s", tools.Slugify(dept), os.Getenv("GROUP_EMAIL_DOMAIN"))
		groupName := fmt.Sprintf("Dept: %s", dept)

		var memberEmails []string
		for _, user := range deptUsers {
			if user.Email != "" {
				memberEmails = append(memberEmails, strings.ToLower(user.Email))
			}
		}

		// 3. Sync to Google Workspace
		svc, err := googleclient.NewDirectoryService(ctx)
		if err != nil {
			tools.Log.WithField("dept", dept).Errorf("Failed to create Google Directory client: %v", err)
			return
		}

		if err := SyncGoogleGroup(ctx, svc, groupEmail, groupName, memberEmails, dryRun); err != nil {
			tools.Log.WithField("dept", dept).Errorf("Google group sync error: %v", err)
		}
	})

	tools.Log.Infof("Finished syncing departments in %s", time.Since(start))
}
