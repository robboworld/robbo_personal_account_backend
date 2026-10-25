package dbmigrate

import (
	"reflect"
	"strings"
	"testing"
)

func TestPendingKeepsOrderAndSkipsApplied(t *testing.T) {
	got := pending([]string{"0001_a.sql", "0002_b.sql", "0003_c.sql"}, []string{"0002_b.sql"})
	want := []string{"0001_a.sql", "0003_c.sql"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending = %v, want %v", got, want)
	}
}

func TestEmbeddedMigrationsAreOrderedSQL(t *testing.T) {
	for _, db := range []string{"licensing", "projects"} {
		list, err := versions(db)
		if err != nil {
			t.Fatalf("%s: %v", db, err)
		}
		for i, v := range list {
			if !strings.HasSuffix(v, ".sql") || len(v) < 5 || v[4] != '_' {
				t.Fatalf("%s: %q is not NNNN_name.sql", db, v)
			}
			if i > 0 && list[i-1] >= v {
				t.Fatalf("%s: %q is not after %q", db, v, list[i-1])
			}
		}
	}
	lic, _ := versions("licensing")
	if len(lic) == 0 || lic[0] != "0001_tariff_limits.sql" {
		t.Fatalf("licensing migrations = %v", lic)
	}
}
