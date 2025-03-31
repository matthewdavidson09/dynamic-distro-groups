package active_directory

import (
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ─── Normalization ───

func NormalizeDN(dn string) string {
	return strings.ToLower(strings.TrimSpace(dn))
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

func GroupUsersByManager(users []ADUser) map[string][]ADUser {
	managerMap := make(map[string][]ADUser)
	for _, user := range users {
		if user.ManagerDN != "" {
			managerMap[NormalizeDN(user.ManagerDN)] = append(managerMap[NormalizeDN(user.ManagerDN)], user)
		}
	}
	return managerMap
}

func mapKeysSorted(m map[string]struct{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
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
