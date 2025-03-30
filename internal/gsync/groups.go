package gsync

import (
	"context"
	"fmt"

	"github.com/matthewdavidson09/dynamic-distro-groups/internal/googleclient"
)

func SyncGroupsFromAD() error {
	ctx := context.Background()
	svc, err := googleclient.NewDirectoryService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create directory service: %w", err)
	}

	groups, err := svc.Groups.List().Customer("my_customer").MaxResults(1000).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch Google Groups: %w", err)
	}

	for _, g := range groups.Groups {
		membersList, err := svc.Members.List(g.Id).Do()
		if err != nil {
			fmt.Printf("📧 %s | ID: %s | Members: error: %v\n", g.Email, g.Id, err)
			continue
		}

		memberCount := len(membersList.Members)
		fmt.Printf("📧 %s | ID: %s | Members: %d\n", g.Email, g.Id, memberCount)
	}

	return nil
}
