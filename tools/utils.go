package tools

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FormatGUID converts a raw objectGUID []byte into a standard Microsoft GUID string
func FormatGUID(b []byte) string {
	if len(b) != 16 {
		return ""
	}
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[3], b[2], b[1], b[0],
		b[5], b[4],
		b[7], b[6],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	)
}

func IsAccountEnabled(uac string) string {
	if uac == "" {
		return "unknown"
	}
	val, err := strconv.Atoi(uac)
	if err != nil {
		return "unknown"
	}
	if val&0x2 == 0 {
		return "true"
	}
	return "false"
}

func DecodeUserAccountControlFlags(uac string) []string {
	flags := map[int]string{
		0x0001:     "SCRIPT",
		0x0002:     "ACCOUNTDISABLE",
		0x0008:     "HOMEDIR_REQUIRED",
		0x0010:     "LOCKOUT",
		0x0020:     "PASSWD_NOTREQD",
		0x0040:     "PASSWD_CANT_CHANGE", // Not reliable on modern systems
		0x0080:     "ENCRYPTED_TEXT_PASSWORD_ALLOWED",
		0x0100:     "TEMP_DUPLICATE_ACCOUNT",
		0x0200:     "NORMAL_ACCOUNT",
		0x0800:     "INTERDOMAIN_TRUST_ACCOUNT",
		0x1000:     "WORKSTATION_TRUST_ACCOUNT",
		0x2000:     "SERVER_TRUST_ACCOUNT",
		0x10000:    "DONT_EXPIRE_PASSWORD",
		0x20000:    "MNS_LOGON_ACCOUNT",
		0x40000:    "SMARTCARD_REQUIRED",
		0x80000:    "TRUSTED_FOR_DELEGATION",
		0x100000:   "NOT_DELEGATED",
		0x200000:   "USE_DES_KEY_ONLY",
		0x400000:   "DONT_REQ_PREAUTH",
		0x800000:   "PASSWORD_EXPIRED",
		0x1000000:  "TRUSTED_TO_AUTH_FOR_DELEGATION",
		0x04000000: "PARTIAL_SECRETS_ACCOUNT",
	}

	var activeFlags []string
	val, err := strconv.Atoi(uac)
	if err != nil {
		return []string{"invalid"}
	}

	for bit, label := range flags {
		if val&bit != 0 {
			activeFlags = append(activeFlags, label)
		}
	}

	return activeFlags
}

// slugify converts names like "Human Resources" to "human-resources"
func Slugify(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))

	// Replace spaces and underscores with dashes
	input = strings.ReplaceAll(input, " ", "-")
	input = strings.ReplaceAll(input, "_", "-")

	// Remove all non-alphanumeric or dash characters
	re := regexp.MustCompile(`[^a-z0-9\-]`)
	input = re.ReplaceAllString(input, "")

	// Collapse multiple dashes
	reDash := regexp.MustCompile(`-+`)
	input = reDash.ReplaceAllString(input, "-")

	// Trim leading/trailing dashes
	input = strings.Trim(input, "-")

	return input
}

// MapKeys returns a slice of keys from a map[string]T
func MapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
