package active_directory

import (
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

type SyncResult struct {
	Label    string
	Added    int
	Removed  int
	Duration time.Duration
	Success  bool
	Error    error
}

// SyncGroupByCategory ensures a group exists and syncs its members.
func SyncGroupByCategory(client *ldapclient.LDAPClient, category, value string, users []ADUser, dryRun bool) (int, int, error) {
	slug := tools.Slugify(value)
	groupCN := fmt.Sprintf("list-%s-%s", category, slug)
	email := fmt.Sprintf("%s@giftingco.com", groupCN)
	groupOU := "OU=Automated Groups,OU=Groups,DC=corp,DC=agiftinside,DC=com"

	group, err := EnsureGroupExists(client, groupCN, email, groupOU, value)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to ensure group: %w", err)
	}

	// Always ensure the mail attribute is correct
	if mailErr := EnsureGroupMailAttribute(client, group.DN, email); mailErr != nil {
		tools.Log.WithError(mailErr).Warnf("Could not update mail attribute for %s", group.DN)
	}

	added, removed, err := syncGroupMembers(client, group, users, groupCN, dryRun)
	return added, removed, err
}

// AddUserToGroup adds a user (by DN) to the group's "member" attribute.
func AddUserToGroup(client *ldapclient.LDAPClient, groupDN, userDN string) error {
	modReq := ldap.NewModifyRequest(groupDN, nil)
	modReq.Add("member", []string{userDN})

	if err := client.Conn.Modify(modReq); err != nil {
		return fmt.Errorf("failed to add user %s to group %s: %w", userDN, groupDN, err)
	}

	return nil
}

// RemoveUserFromGroup removes a user (by DN) from the group's "member" attribute.
func RemoveUserFromGroup(client *ldapclient.LDAPClient, groupDN, userDN string) error {
	modReq := ldap.NewModifyRequest(groupDN, nil)
	modReq.Delete("member", []string{userDN})

	if err := client.Conn.Modify(modReq); err != nil {
		return fmt.Errorf("failed to remove user %s from group %s: %w", userDN, groupDN, err)
	}

	return nil
}

// syncGroupMembers calculates and applies the difference between desired and actual group members.
func syncGroupMembers(client *ldapclient.LDAPClient, group *ADGroup, users []ADUser, label string, dryRun bool) (int, int, error) {
	current := make(map[string]struct{})
	for _, dn := range group.Members {
		current[normalizeDN(dn)] = struct{}{}
	}

	desired := make(map[string]ADUser)
	for _, u := range users {
		desired[normalizeDN(u.DN)] = u
	}

	var toAdd, toRemove []string
	for dn := range desired {
		if _, exists := current[dn]; !exists {
			toAdd = append(toAdd, dn)
		}
	}
	for dn := range current {
		if _, exists := desired[dn]; !exists {
			toRemove = append(toRemove, dn)
		}
	}

	tools.Log.WithFields(map[string]interface{}{
		"group":  label,
		"add":    len(toAdd),
		"remove": len(toRemove),
	}).Debug("Sync plan")

	if dryRun {
		tools.Log.Debug("Dry run enabled — no changes made.")
		for _, dn := range toAdd {
			tools.Log.Debugf("[DRY] Add %s → %s", dn, label)
		}
		for _, dn := range toRemove {
			tools.Log.Debugf("[DRY] Remove %s ← %s", dn, label)
		}
		return len(toAdd), len(toRemove), nil
	}

	for _, dn := range toAdd {
		tools.Log.Debugf("Adding %s → %s", dn, label)
		if err := AddUserToGroup(client, group.DN, dn); err != nil {
			tools.Log.WithError(err).Errorf("Failed to add %s", dn)
		}
	}

	for _, dn := range toRemove {
		tools.Log.Debugf("Removing %s ← %s", dn, label)
		if err := RemoveUserFromGroup(client, group.DN, dn); err != nil {
			tools.Log.WithError(err).Errorf("Failed to remove %s", dn)
		}
	}

	return len(toAdd), len(toRemove), nil
}
