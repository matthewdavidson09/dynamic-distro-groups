package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
	admin "google.golang.org/api/admin/directory/v1"
)

// SyncGoogleGroup ensures a Google Group exists and reflects the given AD users
func SyncGoogleGroup(ctx context.Context, svc *admin.Service, groupEmail string, groupName string, adUsers []string, dryRun bool) error {
	group, err := getOrCreateGoogleGroup(ctx, svc, groupEmail, groupName)
	if err != nil {
		return fmt.Errorf("google group sync error: %w", err)
	}

	adUserSet := make(map[string]struct{})
	userCache := make(map[string]bool)
	for _, email := range adUsers {
		adUserSet[email] = struct{}{}
	}

	// Fetch current members
	currentMembers := make(map[string]bool)
	memberList, err := svc.Members.List(group.Email).Do()
	if err == nil {
		for _, m := range memberList.Members {
			email := strings.ToLower(strings.TrimSpace(m.Email))
			currentMembers[email] = true
		}
	}

	// Determine changes
	var toAdd, toRemove []string

	for email := range adUserSet {
		email = strings.ToLower(strings.TrimSpace(email))

		if email == "" || currentMembers[email] {
			continue
		}
		if allowed, cached := userCache[email]; cached {
			if allowed {
				toAdd = append(toAdd, email)
			}
			continue
		}
		allowed := isMailboxUser(svc, email)
		userCache[email] = allowed
		if allowed {
			toAdd = append(toAdd, email)
		} else {
			tools.Log.Debugf("Skipping %s — no mailbox", email)
		}
	}

	for email := range currentMembers {
		if _, shouldBePresent := adUserSet[email]; !shouldBePresent {
			toRemove = append(toRemove, email)
		}
	}

	tools.Log.WithFields(map[string]interface{}{
		"group":   groupEmail,
		"add":     len(toAdd),
		"remove":  len(toRemove),
		"dry_run": dryRun,
	}).Info("Google Group sync plan")

	if dryRun {
		for _, email := range toAdd {
			tools.Log.Infof("[DRY RUN] Would add %s to %s", email, groupEmail)
		}
		for _, email := range toRemove {
			tools.Log.Infof("[DRY RUN] Would remove %s from %s", email, groupEmail)
		}
		return nil
	}

	// Apply changes
	for _, email := range toAdd {
		member := &admin.Member{Email: email, Role: "MEMBER"}
		if _, err := svc.Members.Insert(group.Email, member).Do(); err != nil {
			tools.Log.WithError(err).Errorf("Failed to add %s to %s", email, groupEmail)
		} else {
			tools.Log.Infof("Added %s to %s", email, groupEmail)
		}
	}

	for _, email := range toRemove {
		if err := svc.Members.Delete(group.Email, email).Do(); err != nil {
			tools.Log.WithError(err).Errorf("Failed to remove %s from %s", email, groupEmail)
		} else {
			tools.Log.Infof("Removed %s from %s", email, groupEmail)
		}
	}

	return nil
}
