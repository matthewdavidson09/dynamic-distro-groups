package sync

import (
	"context"
	"fmt"

	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/googleapi"
)

func getOrCreateGoogleGroup(ctx context.Context, svc *admin.Service, email, name string) (*admin.Group, error) {
	group, err := svc.Groups.Get(email).Do()
	if err == nil {
		return group, nil
	}

	// Try creating the group if it doesn’t exist
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 404 {
		group := &admin.Group{
			Email:       email,
			Name:        name,
			Description: "Synced from Active Directory",
		}
		created, err := svc.Groups.Insert(group).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to create group %s: %w", email, err)
		}
		if created == nil {
			return nil, fmt.Errorf("group is nil after creation: %s", email)
		}
		return created, nil
	}

	return nil, fmt.Errorf("failed to get group %s: %w", email, err)
}

// isMailboxUser returns true if Gmail is enabled for this user
func isMailboxUser(svc *admin.Service, email string) bool {
	user, err := svc.Users.Get(email).Do()
	if err != nil {
		tools.Log.Debugf("Failed user lookup for %s: %v", email, err)
		return false
	}
	return user.IsMailboxSetup
}
