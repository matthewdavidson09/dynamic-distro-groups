package active_directory

import (
	"slices"
	"strings"

	"github.com/matthewdavidson09/dynamic-distro-groups/internal/ldapclient"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ─── Normalization ───

func normalizeDN(dn string) string {
	return strings.ToLower(strings.TrimSpace(dn))
}

func normalizeOU(ou string) string {
	return strings.ToLower(strings.TrimSpace(ou))
}

// ─── Grouping Utilities ───

func GetUniqueStates(users []ADUser) []string {
	stateMap := make(map[string]struct{})
	for _, user := range users {
		state := strings.ToUpper(strings.TrimSpace(user.State))
		if state != "" {
			stateMap[state] = struct{}{}
		}
	}
	return mapKeysSorted(stateMap)
}

func GetUniqueDepartments(users []ADUser) []string {
	caser := cases.Title(language.English)
	deptMap := make(map[string]struct{})
	for _, user := range users {
		dept := caser.String(strings.ToLower(strings.TrimSpace(user.Department)))
		if dept != "" {
			deptMap[dept] = struct{}{}
		}
	}
	return mapKeysSorted(deptMap)
}

func mapKeysSorted(m map[string]struct{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// ─── Public Fetch Helpers ───

func GetAllUniqueStates(enabledOnly bool) ([]string, error) {
	client, err := ldapclient.Connect()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	users, err := GetUsersByFilter(client, nil, enabledOnly, true, defaultOUsToExclude())
	if err != nil {
		return nil, err
	}

	return GetUniqueStates(users), nil
}

func GetAllEmployees(client *ldapclient.LDAPClient) ([]ADUser, error) {
	return GetUsersByFilter(client, nil, true, true, defaultOUsToExclude())
}

func defaultOUsToExclude() []string {
	return []string{"OU=External Users", "OU=Archived Users"}
}

func isUserDisabled(uac string) bool {
	return strings.Contains(uac, "2")
}

func parseUACFlags(uac string) []string {
	if uac == "" {
		return nil
	}
	return strings.Split(uac, ",")
}

func shouldExcludeOU(dn string, excludeOUs []string) bool {
	lowerDN := strings.ToLower(dn)
	for _, ou := range excludeOUs {
		if strings.Contains(lowerDN, strings.ToLower(ou)) {
			return true
		}
	}
	return false
}
