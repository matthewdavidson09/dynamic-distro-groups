package active_directory

import (
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

type ADGroup struct {
	CN         string
	DN         string
	Email      string
	Members    []string
	ObjectGUID string
}

func GetGroupByEmail(client *ldapclient.LDAPClient, email, baseDN string) (*ADGroup, error) {
	filter := fmt.Sprintf("(mail=%s)", ldap.EscapeFilter(email))
	attributes := []string{"cn", "distinguishedName", "mail", "member", "objectGUID"}

	searchReq := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeSingleLevel,
		ldap.NeverDerefAliases,
		1, 0, false,
		filter,
		attributes,
		nil,
	)

	result, err := client.Conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("LDAP search error: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("group not found with email: %s", email)
	}

	entry := result.Entries[0]
	return &ADGroup{
		CN:         entry.GetAttributeValue("cn"),
		DN:         entry.DN,
		Email:      entry.GetAttributeValue("mail"),
		Members:    entry.GetAttributeValues("member"),
		ObjectGUID: tools.FormatGUID(entry.GetRawAttributeValue("objectGUID")),
	}, nil
}

func GetGroupByCN(client *ldapclient.LDAPClient, cn, baseDN string) (*ADGroup, error) {
	filter := fmt.Sprintf("(cn=%s)", ldap.EscapeFilter(cn))
	attributes := []string{"cn", "distinguishedName", "mail", "member", "objectGUID"}

	searchReq := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeSingleLevel,
		ldap.NeverDerefAliases,
		1, 0, false,
		filter,
		attributes,
		nil,
	)

	result, err := client.Conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("LDAP search error: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("group not found with CN: %s", cn)
	}

	entry := result.Entries[0]
	return &ADGroup{
		CN:         entry.GetAttributeValue("cn"),
		DN:         entry.DN,
		Email:      entry.GetAttributeValue("mail"),
		Members:    entry.GetAttributeValues("member"),
		ObjectGUID: tools.FormatGUID(entry.GetRawAttributeValue("objectGUID")),
	}, nil
}

func CreateGroup(client *ldapclient.LDAPClient, cn, email, ou, dept string) error {
	groupDN := fmt.Sprintf("CN=%s,%s", cn, ou)
	label := fmt.Sprintf("All %s Employees", dept)

	addReq := ldap.NewAddRequest(groupDN, nil)
	addReq.Attribute("objectClass", []string{"top", "group"})
	addReq.Attribute("cn", []string{cn})
	addReq.Attribute("sAMAccountName", []string{cn})
	addReq.Attribute("mail", []string{email})
	addReq.Attribute("displayName", []string{label})
	addReq.Attribute("description", []string{label + " distro group"})
	addReq.Attribute("groupType", []string{fmt.Sprint(0x00000008)})

	err := client.Conn.Add(addReq)
	if err != nil {
		tools.Log.WithFields(map[string]interface{}{
			"dn":    groupDN,
			"error": err,
		}).Error("Failed to create group")
		return fmt.Errorf("failed to create group: %w", err)
	}

	tools.Log.WithField("cn", cn).Info("Group created successfully")
	return nil
}

func EnsureGroupMailAttribute(client *ldapclient.LDAPClient, groupDN, expectedEmail string) error {
	searchReq := ldap.NewSearchRequest(
		groupDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1, 0, false,
		"(objectClass=group)",
		[]string{"mail"},
		nil,
	)

	result, err := client.Conn.Search(searchReq)
	if err != nil {
		return fmt.Errorf("failed to search for group mail attribute: %w", err)
	}
	if len(result.Entries) == 0 {
		return fmt.Errorf("group not found at DN: %s", groupDN)
	}

	currentMail := result.Entries[0].GetAttributeValue("mail")
	if currentMail == expectedEmail {
		tools.Log.WithField("dn", groupDN).Debug("Mail attribute already correct")
		return nil
	}

	modReq := ldap.NewModifyRequest(groupDN, nil)
	if currentMail != "" {
		modReq.Replace("mail", []string{expectedEmail})
		tools.Log.WithFields(map[string]interface{}{
			"dn":    groupDN,
			"email": expectedEmail,
		}).Info("Replacing existing mail attribute")
	} else {
		modReq.Add("mail", []string{expectedEmail})
		tools.Log.WithFields(map[string]interface{}{
			"dn":    groupDN,
			"email": expectedEmail,
		}).Info("Adding mail attribute")
	}

	if err := client.Conn.Modify(modReq); err != nil {
		return fmt.Errorf("failed to update mail attribute: %w", err)
	}

	return nil
}
