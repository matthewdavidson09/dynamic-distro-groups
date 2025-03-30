package main

import (
	"time"

	"github.com/joho/godotenv"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/active_directory"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/sync"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

func main() {
	// Load environment and init logger
	if err := godotenv.Load(".env"); err != nil {
		tools.Log.Fatalf("Failed to load .env file: %v", err)
	}
	tools.InitLogger()

	dryRun := false // Set to true to skip modifying LDAP

	// Connect to LDAP
	client, err := ldapclient.Connect()
	if err != nil {
		tools.Log.Fatalf("Failed to connect to LDAP: %v", err)
	}
	defer client.Close()

	// Load all eligible users once
	allUsers, err := active_directory.GetUsersByFilter(
		client,
		nil,  // No custom filter map
		true, // Only enabled users
		true, // Require mail attribute
		[]string{"OU=External Users", "OU=Archived Users"}, // Excluded OUs
	)
	if err != nil {
		tools.Log.Fatalf("Failed to fetch users: %v", err)
	}

	// Sync by department
	start := time.Now()
	sync.RunAllGroupSyncs(client, allUsers, dryRun)
	tools.Log.Infof("Finished syncing all groups in %s", time.Since(start))
}
