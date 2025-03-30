package active_directory

import (
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

// ADUser represents a simplified Active Directory user object
type ADUser struct {
	CN             string
	DN             string
	GUID           string
	DisplayName    string
	GivenName      string
	Surname        string
	Email          string
	EmployeeID     string
	Department     string
	Title          string
	StreetAddress  string
	City           string
	State          string
	PostalCode     string
	ManagerDN      string
	SAMAccountName string
	Enabled        bool
	UACFlags       []string
	DirectReports  []string
}

// GetUsersByFilter returns a list of AD users based on the provided filter and criteria
func GetUsersByFilter(
	client *ldapclient.LDAPClient,
	filterMap map[string]string,
	enabledOnly bool,
	requireMail bool,
	excludeOUs []string,
) ([]ADUser, error) {
	var filterParts []string
	filterParts = append(filterParts, "(objectClass=user)")

	if enabledOnly {
		filterParts = append(filterParts, "(!(userAccountControl:1.2.840.113556.1.4.803:=2))")
	}

	for attr, value := range filterMap {
		if strings.HasPrefix(attr, "!") || strings.HasSuffix(attr, "=*") {
			// Pass-through custom LDAP filter fragments like "!(lastLogon=*)"
			filterParts = append(filterParts, attr)
		} else if value == "" {
			// Assume user means attribute must not exist
			filterParts = append(filterParts, fmt.Sprintf("(!(%s=*))", ldap.EscapeFilter(attr)))
		} else {
			filterParts = append(filterParts, fmt.Sprintf("(%s=%s)", ldap.EscapeFilter(attr), ldap.EscapeFilter(value)))
		}
	}

	ldapFilter := fmt.Sprintf("(&%s)", strings.Join(filterParts, ""))

	attributes := []string{
		"cn", "mail", "department", "distinguishedName", "st", "userAccountControl",
		"objectGUID", "givenName", "sn", "displayName", "employeeID", "title",
		"streetAddress", "l", "postalCode", "manager", "sAMAccountName", "directReports",
	}

	searchReq := ldap.NewSearchRequest(
		client.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		ldapFilter,
		attributes,
		nil,
	)

	result, err := client.Conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	var users []ADUser
	for _, entry := range result.Entries {
		dn := entry.GetAttributeValue("distinguishedName")

		if shouldExcludeOU(dn, excludeOUs) {
			continue
		}

		email := entry.GetAttributeValue("mail")
		if requireMail && email == "" {
			continue
		}

		users = append(users, ADUser{
			CN:             entry.GetAttributeValue("cn"),
			DN:             dn,
			GUID:           tools.FormatGUID(entry.GetRawAttributeValue("objectGUID")),
			DisplayName:    entry.GetAttributeValue("displayName"),
			GivenName:      entry.GetAttributeValue("givenName"),
			Surname:        entry.GetAttributeValue("sn"),
			Email:          email,
			EmployeeID:     entry.GetAttributeValue("employeeID"),
			Department:     entry.GetAttributeValue("department"),
			Title:          entry.GetAttributeValue("title"),
			StreetAddress:  entry.GetAttributeValue("streetAddress"),
			City:           entry.GetAttributeValue("l"), // 'l' is LDAP attribute for 'city'
			State:          entry.GetAttributeValue("st"),
			PostalCode:     entry.GetAttributeValue("postalCode"),
			ManagerDN:      entry.GetAttributeValue("manager"),
			DirectReports:  entry.GetAttributeValues("directReports"),
			SAMAccountName: entry.GetAttributeValue("sAMAccountName"),
			Enabled:        !isUserDisabled(entry.GetAttributeValue("userAccountControl")),
			UACFlags:       parseUACFlags(entry.GetAttributeValue("userAccountControl")),
		})
	}

	return users, nil
}
