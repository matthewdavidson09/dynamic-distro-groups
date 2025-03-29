module github.com/matthewdavidson09/dynamic-distro-groups

go 1.23.0

toolchain go1.23.7

require (
	github.com/go-ldap/ldap/v3 v3.4.10
	github.com/joho/godotenv v1.5.1
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/text v0.23.0
)

require (
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.7 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
)

replace github.com/matthewdavidson09/dynamic-distro-groups => ./
