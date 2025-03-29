package active_directory

import (
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

var ErrGroupNotFound = errors.New("group not found")

func EnsureGroupExists(client *ldapclient.LDAPClient, cn, email, ou, label string) (*ADGroup, error) {
	tools.Log.WithFields(map[string]interface{}{
		"cn":    cn,
		"email": email,
	}).Debug("Ensuring group exists")

	// 1. Try to fetch by email
	group, err := GetGroupByEmail(client, email, ou)
	if err == nil {
		tools.Log.WithField("cn", cn).Debug("Group found by email")
		return group, nil
	}

	// 2. If not found, try by CN
	group, errCN := GetGroupByCN(client, cn, ou)
	if errCN == nil {
		tools.Log.WithField("cn", cn).Debug("Group found by CN (mail may be missing)")
		if mailErr := EnsureGroupMailAttribute(client, group.DN, email); mailErr != nil {
			tools.Log.WithFields(map[string]interface{}{
				"cn":    cn,
				"error": mailErr,
			}).Warn("Failed to update mail attribute")
		}
		return group, nil
	}

	// 3. Otherwise, create it
	tools.Log.WithFields(map[string]interface{}{
		"cn":    cn,
		"email": email,
	}).Info("Group not found, creating new group")

	if err := CreateGroup(client, cn, email, ou, label); err != nil {
		if ldapErr, ok := err.(*ldap.Error); ok && ldapErr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
			tools.Log.WithField("cn", cn).Warn("Group already created by another process. Retrying fetch...")
		} else {
			return nil, fmt.Errorf("failed to create group: %w", err)
		}
	}

	// 4. Fetch again
	group, err = GetGroupByEmail(client, email, ou)
	if err != nil {
		return nil, fmt.Errorf("group created but cannot be fetched: %w", err)
	}

	tools.Log.WithField("cn", cn).Info("Group creation confirmed")
	return group, nil
}
