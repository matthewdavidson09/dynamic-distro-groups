package main

import (
	"net"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	directory "github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		tools.Log.Fatalf("Failed to load .env file: %v", err)
	}
	tools.InitLogger()

	ldapHost := os.Getenv("LDAP_SERVER")
	ldapPort := os.Getenv("LDAP_PORT")
	if ldapPort == "" {
		ldapPort = "389"
	}

	dryRun := false // Set to true if you want to perform a dry run without making changes

	addrs, err := net.LookupHost(ldapHost)
	if err != nil || len(addrs) == 0 {
		tools.Log.Fatalf("Failed to resolve LDAP server: %v", err)
	}
	resolvedIP := addrs[0]
	tools.Log.Infof("Using LDAP server: %s (%s:%s)", ldapHost, resolvedIP, ldapPort)

	client, err := ldapclient.ConnectWithIP(resolvedIP, ldapPort)
	if err != nil {
		tools.Log.Fatalf("Failed to connect to LDAP: %v", err)
	}
	defer client.Close()

	// Load all enabled + valid users
	allUsers, err := directory.GetUsersByFilter(client, nil, true, true, nil)
	if err != nil {
		tools.Log.Fatalf("Failed to get all users: %v", err)
	}

	departments := directory.GetUniqueDepartments(allUsers)
	states := directory.GetUniqueStates(allUsers)

	// ───── Sync Departments ─────
	start := time.Now()
	tools.Log.Infof("Syncing %d department-based groups...", len(departments))

	tools.RunWithWorkers(departments, 5, func(dept string) {
		var deptUsers []directory.ADUser
		for _, user := range allUsers {
			if strings.EqualFold(strings.TrimSpace(user.Department), dept) {
				deptUsers = append(deptUsers, user)
			}
		}
		if len(deptUsers) == 0 {
			tools.Log.WithField("dept", dept).Warn("No users found, skipping.")
			return
		}

		added, removed, err := directory.SyncGroupByCategory(client, "dept", dept, deptUsers, dryRun)
		tools.LogSyncSummary("dept", dept, len(deptUsers), added, removed)
		if err != nil {
			tools.Log.WithField("dept", dept).Errorf("Sync error: %v", err)
		}
	})
	tools.Log.Infof("Finished syncing departments in %s", time.Since(start))

	// ───── Sync States ─────
	startStates := time.Now()
	tools.Log.Infof("Syncing %d state-based groups...", len(states))

	tools.RunWithWorkers(states, 5, func(state string) {
		var stateUsers []directory.ADUser
		for _, user := range allUsers {
			if strings.EqualFold(strings.TrimSpace(user.State), state) {
				stateUsers = append(stateUsers, user)
			}
		}
		if len(stateUsers) == 0 {
			tools.Log.WithField("state", state).Warn("No users found, skipping.")
			return
		}

		added, removed, err := directory.SyncGroupByCategory(client, "state", state, stateUsers, dryRun)
		tools.LogSyncSummary("state", state, len(stateUsers), added, removed)
		if err != nil {
			tools.Log.WithField("state", state).Errorf("Sync error: %v", err)
		}
	})
	tools.Log.Infof("Finished syncing states in %s", time.Since(startStates))

	// ───── Sync All Employees Group ─────
	tools.Log.Info("Syncing All Employees distribution list...")
	tools.Log.Infof("Found %d eligible employees", len(allUsers))

	added, removed, err := directory.SyncGroupByCategory(client, "all", "employees", allUsers, dryRun)
	if err != nil {
		tools.Log.Errorf("Failed to sync All Employees group: %v", err)
	} else {
		tools.LogSyncSummary("all", "employees", len(allUsers), added, removed)
	}
}
