package main

import "testing"

func TestDemoSeedConsistency(t *testing.T) {
	ledger := seedDemo()

	if got := len(ledger.Spaces); got != 18 {
		t.Fatalf("spaces = %d, want 18", got)
	}
	if got := len(ledger.Equipment); got != 28 {
		t.Fatalf("equipment = %d, want 28", got)
	}
	if got := len(ledger.Projects); got != 1 {
		t.Fatalf("projects = %d, want 1", got)
	}
	if got := len(ledger.Borrows); got != 3 {
		t.Fatalf("borrows = %d, want 3", got)
	}

	assertUniqueIDs(t, "space", spaceIDs(ledger.Spaces))
	assertUniqueIDs(t, "equipment", equipmentIDs(ledger.Equipment))
	assertUniqueIDs(t, "project", projectIDs(ledger.Projects))
	assertUniqueIDs(t, "borrow", borrowIDs(ledger.Borrows))

	if ledger.Stats.SpaceTotal != len(ledger.Spaces) {
		t.Fatalf("space stats = %d, want %d", ledger.Stats.SpaceTotal, len(ledger.Spaces))
	}
	if ledger.Stats.EquipmentTotal != len(ledger.Equipment) {
		t.Fatalf("equipment stats = %d, want %d", ledger.Stats.EquipmentTotal, len(ledger.Equipment))
	}
	if ledger.Stats.ProjectActive != 1 || ledger.Stats.AppPending != 1 || ledger.Stats.BorrowPending != 1 {
		t.Fatalf("unexpected workflow stats: %+v", ledger.Stats)
	}

	spaces := make(map[string]bool, len(ledger.Spaces))
	for _, space := range ledger.Spaces {
		spaces[space.ID] = true
	}
	projects := make(map[string]bool, len(ledger.Projects))
	for _, project := range ledger.Projects {
		projects[project.ID] = true
	}
	for _, allocation := range ledger.Allocations {
		if !spaces[allocation.SpaceID] {
			t.Errorf("allocation %s references unknown space %s", allocation.ID, allocation.SpaceID)
		}
		if !projects[allocation.ProjectID] {
			t.Errorf("allocation %s references unknown project %s", allocation.ID, allocation.ProjectID)
		}
	}
}

func TestDemoAccountsAndSessions(t *testing.T) {
	auth := NewAuthService()

	if _, err := auth.Login("admin", "wrong-password"); err == nil {
		t.Fatal("invalid password unexpectedly succeeded")
	}

	admin, err := auth.Login(" ADMIN ", "xlab2026")
	if err != nil {
		t.Fatalf("admin login failed: %v", err)
	}
	if admin.User.Role != RoleAdmin || admin.User.Org != "Org1MSP" {
		t.Fatalf("unexpected admin identity: %+v", admin.User)
	}
	if _, ok := auth.Resolve(admin.Token); !ok {
		t.Fatal("new session cannot be resolved")
	}
	auth.Logout(admin.Token)
	if _, ok := auth.Resolve(admin.Token); ok {
		t.Fatal("logged-out session still resolves")
	}

	member, err := auth.Login("team", "xlab2026")
	if err != nil {
		t.Fatalf("team login failed: %v", err)
	}
	if member.User.Role != RoleMember || member.User.Org != "Org2MSP" {
		t.Fatalf("unexpected member identity: %+v", member.User)
	}
}

func assertUniqueIDs(t *testing.T, kind string, ids []string) {
	t.Helper()
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" {
			t.Errorf("%s contains an empty id", kind)
		}
		if seen[id] {
			t.Errorf("%s contains duplicate id %s", kind, id)
		}
		seen[id] = true
	}
}

func spaceIDs(items []Space) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func equipmentIDs(items []Equipment) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func projectIDs(items []Project) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func borrowIDs(items []BorrowRecord) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
