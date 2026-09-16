package store

import (
	"path/filepath"
	"testing"

	"jade-gamble/backend/domain"
)

func openTestDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestUserAndChips(t *testing.T) {
	s := openTestDB(t)
	u, err := s.CreateUser("discord123", "tester", "av.png")
	if err != nil {
		t.Fatal(err)
	}
	if u.Chips != domain.SignupChips {
		t.Fatalf("signup bonus: %d", u.Chips)
	}
	got, err := s.GetUserByDiscordID("discord123")
	if err != nil || got.ID != u.ID {
		t.Fatalf("fetch by discord id: %v %v", got, err)
	}

	// spend chips
	if err := s.withTx(func(tx *txWrap) error {
		_, err := UpdateChipsTx(tx, u.ID, -5000)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	got2, _ := s.GetUser(u.ID)
	if got2.Chips != domain.SignupChips-5000 {
		t.Fatalf("after spend: %d", got2.Chips)
	}

	// cannot go negative
	err = s.withTx(func(tx *txWrap) error {
		_, err := UpdateChipsTx(tx, u.ID, -(domain.SignupChips - 5000 + 1000))
		return err
	})
	if err != domain.ErrInsufficientChips {
		t.Fatalf("expected ErrInsufficientChips, got %v", err)
	}
}

func TestStoneRoundtrip(t *testing.T) {
	s := openTestDB(t)
	u, _ := s.CreateUser("d2", "u2", "")
	st := &domain.Stone{
		ID: "S_abc", Seed: 12345, Grade: domain.KiloGrade, Price: 800,
		Quality: domain.Bean, Variety: domain.Violet,
		CrackCells: []int{3, 9}, CracksDeep: true, InkHidden: false,
		LightHint: "test hint", LightLieRate: 0.3, OwnerID: u.ID, State: domain.StateOwned,
	}
	if err := s.SaveStone(st); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetStone("S_abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.Quality != domain.Bean || got.Variety != domain.Violet || got.Price != 800 ||
		len(got.CrackCells) != 2 || !got.CracksDeep || got.OwnerID != u.ID {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if got.LightHint != "test hint" {
		t.Fatalf("hint: %q", got.LightHint)
	}
}

func TestSession(t *testing.T) {
	s := openTestDB(t)
	u, _ := s.CreateUser("d3", "u3", "")
	tok, err := s.CreateSession(u.ID)
	if err != nil || len(tok) != 64 {
		t.Fatalf("session: %v %v", tok, err)
	}
	uid, err := s.GetSession(tok)
	if err != nil || uid != u.ID {
		t.Fatalf("lookup: %v %v", uid, err)
	}
	if err := s.DeleteSession(tok); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSession(tok); err == nil {
		t.Fatal("deleted session still resolves")
	}
}

func TestDiscovery(t *testing.T) {
	s := openTestDB(t)
	u, _ := s.CreateUser("d4", "u4", "")
	first, err := s.withTxBool(func(tx *txWrap) (bool, error) {
		return s.DiscoverTx(tx, u.ID, domain.Violet)
	})
	if err != nil || !first {
		t.Fatalf("first discovery: %v %v", first, err)
	}
	second, _ := s.withTxBool(func(tx *txWrap) (bool, error) {
		return s.DiscoverTx(tx, u.ID, domain.Violet)
	})
	if second {
		t.Fatal("duplicate discovery reported as first")
	}
	list, _ := s.DiscoveredList(u.ID)
	if len(list) != 1 || list[0] != domain.Violet {
		t.Fatalf("list: %v", list)
	}
}
