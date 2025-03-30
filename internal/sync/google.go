package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
	admin "google.golang.org/api/admin/directory/v1"
)

// normalizeEmail lowercases and trims whitespace for reliable comparisons.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// SyncGoogleGroup ensures a Google Group exists and reflects the given AD users.
func SyncGoogleGroup(
	ctx context.Context,
	svc *admin.Service,
	groupEmail, groupName string,
	adUsers []string,
	dryRun bool,
) (int, int, error) {
	group, err := getOrCreateGoogleGroup(ctx, svc, groupEmail, groupName)
	if err != nil {
		return 0, 0, fmt.Errorf("google group sync error: %w", err)
	}

	adUserSet := make(map[string]struct{})
	userCache := make(map[string]bool)

	for _, email := range adUsers {
		normalized := normalizeEmail(email)
		if normalized != "" {
			adUserSet[normalized] = struct{}{}
		}
	}

	// Fetch current Google Group members (paged)
	currentMembers := make(map[string]bool)
	memberListCall := svc.Members.List(group.Email)
	err = memberListCall.Pages(ctx, func(page *admin.Members) error {
		for _, m := range page.Members {
			if m.Email != "" {
				currentMembers[normalizeEmail(m.Email)] = true
			}
			if m.Id != "" && m.Type == "ALIAS" {
				currentMembers[normalizeEmail(m.Id)] = true
			}
		}
		return nil
	})
	if err != nil {
		tools.Log.WithError(err).Errorf("Error fetching group members for %s", group.Email)
	}

	var toAdd, toRemove []string

	// Determine additions
	for email := range adUserSet {
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

	// Determine removals
	for email := range currentMembers {
		if _, ok := adUserSet[email]; !ok {
			toRemove = append(toRemove, email)
		}
	}

	if dryRun {
		for _, email := range toAdd {
			tools.Log.Infof("[DRY RUN] Would add %s to %s", email, groupEmail)
		}
		for _, email := range toRemove {
			tools.Log.Infof("[DRY RUN] Would remove %s from %s", email, groupEmail)
		}
		return len(toAdd), len(toRemove), nil
	}

	// Apply additions
	for _, email := range toAdd {
		member := &admin.Member{Email: email, Role: "MEMBER"}
		_, err := svc.Members.Insert(group.Email, member).Do()
		if err != nil {
			tools.Log.WithError(err).Errorf("Failed to add %s to %s", email, groupEmail)
		} else {
			tools.Log.Infof("Added %s to %s", email, groupEmail)
		}
	}

	// Apply removals
	for _, email := range toRemove {
		if err := svc.Members.Delete(group.Email, email).Do(); err != nil {
			tools.Log.WithError(err).Errorf("Failed to remove %s from %s", email, groupEmail)
		} else {
			tools.Log.Infof("Removed %s from %s", email, groupEmail)
		}
	}

	return len(toAdd), len(toRemove), nil
}
