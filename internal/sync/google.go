package sync

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/groupssettings/v1"
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

func ApplyGoogleGroupSettings(ctx context.Context, groupEmail string) error {
	settingsService, err := groupssettings.NewService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GroupsSettings service: %w", err)
	}

	settings := &groupssettings.Groups{
		AllowExternalMembers:     "false",
		AllowWebPosting:          "false",
		AllowGoogleCommunication: "false",
		IsArchived:               "true",
		MembersCanPostAsTheGroup: "false",
		ShowInGroupDirectory:     "false",
		MessageModerationLevel:   "MODERATE_NONE",
		WhoCanContactOwner:       "ALL_IN_DOMAIN_CAN_CONTACT",
		WhoCanAdd:                "NONE_CAN_ADD",
		WhoCanDiscoverGroup:      "ALL_MEMBERS_CAN_DISCOVER",
		WhoCanJoin:               "INVITED_CAN_JOIN",
		WhoCanLeaveGroup:         "NONE_CAN_LEAVE",
		WhoCanPostMessage:        "ALL_MANAGERS_CAN_POST",
		WhoCanViewGroup:          "ALL_MEMBERS_CAN_VIEW",
		WhoCanViewMembership:     "ALL_MANAGERS_CAN_VIEW",
		WhoCanInvite:             "NONE_CAN_INVITE",
		ReplyTo:                  "REPLY_TO_MANAGERS",
	}

	const maxRetries = 5
	var attemptErr error

	for i := 0; i < maxRetries; i++ {
		_, attemptErr = settingsService.Groups.Update(groupEmail, settings).Do()
		if attemptErr == nil {
			return nil
		}

		if strings.Contains(attemptErr.Error(), "Unable to lookup group") || strings.Contains(attemptErr.Error(), "Error 404") {
			// Backoff before retry
			wait := time.Duration(1<<i)*time.Second + time.Duration(rand.Intn(500))*time.Millisecond
			time.Sleep(wait)
			continue
		}

		// Other errors - don't retry
		return fmt.Errorf("failed to apply group settings: %w", attemptErr)
	}

	return fmt.Errorf("failed to apply group settings after %d retries: %w", maxRetries, attemptErr)
}

// SyncGoogleGroupWithRoles syncs users to a Google Group, assigning MANAGER or MEMBER roles.
func SyncGoogleGroupWithRoles(
	ctx context.Context,
	svc *admin.Service,
	groupEmail, groupName string,
	memberEmails []string,
	managerEmails map[string]bool,
	dryRun bool,
) (int, int, error) {
	group, err := getOrCreateGoogleGroup(ctx, svc, groupEmail, groupName)
	if err != nil {
		return 0, 0, fmt.Errorf("get/create group failed: %w", err)
	}

	// Build desired member -> role map
	desiredMembers := map[string]string{}
	userCache := make(map[string]bool)

	for _, email := range memberEmails {
		email = normalizeEmail(email)
		if email == "" {
			continue
		}

		// Cache mailbox check
		allowed, cached := userCache[email]
		if !cached {
			allowed = isMailboxUser(svc, email)
			userCache[email] = allowed
		}
		if !allowed {
			tools.Log.Debugf("Skipping %s — no mailbox setup", email)
			continue
		}

		role := "MEMBER"
		if managerEmails[email] {
			role = "MANAGER"
		}
		desiredMembers[email] = role
	}

	// Get current members (with roles)
	currentMembers := map[string]string{}
	err = svc.Members.List(group.Email).Pages(ctx, func(page *admin.Members) error {
		for _, m := range page.Members {
			if m.Email != "" {
				currentMembers[normalizeEmail(m.Email)] = m.Role
			}
		}
		return nil
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch current members: %w", err)
	}

	var toAdd, toRemove, toUpdate []string

	for email, desiredRole := range desiredMembers {
		currentRole, exists := currentMembers[email]
		if !exists {
			toAdd = append(toAdd, email)
		} else if currentRole != desiredRole {
			toUpdate = append(toUpdate, email)
		}
	}

	for email := range currentMembers {
		if _, ok := desiredMembers[email]; !ok {
			toRemove = append(toRemove, email)
		}
	}

	tools.Log.WithFields(map[string]interface{}{
		"group":   groupEmail,
		"add":     len(toAdd),
		"update":  len(toUpdate),
		"remove":  len(toRemove),
		"dry_run": dryRun,
	}).Debug("Google Group sync plan")

	if dryRun {
		for _, email := range toAdd {
			tools.Log.Infof("[DRY RUN] Would add %s to %s as %s", email, groupEmail, desiredMembers[email])
		}
		for _, email := range toUpdate {
			tools.Log.Infof("[DRY RUN] Would update role for %s to %s", email, desiredMembers[email])
		}
		for _, email := range toRemove {
			tools.Log.Infof("[DRY RUN] Would remove %s from %s", email, groupEmail)
		}
		return len(toAdd), len(toRemove), nil
	}

	// Add new members
	for _, email := range toAdd {
		member := &admin.Member{Email: email, Role: desiredMembers[email]}
		if _, err := svc.Members.Insert(group.Email, member).Do(); err != nil {
			tools.Log.WithError(err).Errorf("Failed to add %s to %s", email, groupEmail)
		} else {
			tools.Log.Infof("Added %s as %s to %s", email, member.Role, groupEmail)
		}
	}

	// Update roles
	for _, email := range toUpdate {
		member := &admin.Member{Role: desiredMembers[email]}
		if _, err := svc.Members.Update(group.Email, email, member).Do(); err != nil {
			tools.Log.WithError(err).Errorf("Failed to update role for %s in %s", email, groupEmail)
		} else {
			tools.Log.Infof("Updated %s to role %s in %s", email, member.Role, groupEmail)
		}
	}

	// Remove obsolete
	for _, email := range toRemove {
		if err := svc.Members.Delete(group.Email, email).Do(); err != nil {
			tools.Log.WithError(err).Errorf("Failed to remove %s from %s", email, groupEmail)
		} else {
			tools.Log.Infof("Removed %s from %s", email, groupEmail)
		}
	}

	return len(toAdd), len(toRemove), nil
}
