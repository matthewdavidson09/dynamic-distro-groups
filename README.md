# Dynamic Distro Groups

A lightweight Go tool for automatically generating and syncing distribution lists (DLs) in Active Directory based on dynamic user attributes such as department, state, or other custom fields. Designed for IT/Infra engineers who want automated, reliable group management without complex overhead.

## ✨ Features

- 🧠 Smart filtering: only includes enabled users with valid email addresses
- 🏢 Supports grouping by:
  - Department
  - State
  - "All Employees" list
  - Future support for Manager, Title, Location, etc.
- ⚡ Fast and concurrent: uses worker pools to parallelize group creation and syncing
- 🧼 Sync logic:
  - Creates groups if missing
  - Ensures group mail attribute
  - Adds/removes users to match the source of truth
- 📦 Configurable via `.env` file
- 🔐 LDAP authentication & connection pooling
- 🪵 Structured logging using `logrus`

## 🛠 Requirements

- Go 1.20+
- Access to your LDAP/AD server
- A service account with permission to:
  - Query users
  - Create/update distribution groups
  - Modify group membership
