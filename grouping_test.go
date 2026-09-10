package dashboard

import (
	"testing"

	health "github.com/larsartmann/go-health"
)

// sourceChecks builds a mixed aggregate-style check set: two sources with
// differing health plus one plain un-namespaced key.
func sourceChecks() map[string]health.Check {
	return map[string]health.Check{
		"api/postgres":      {Status: health.StatusPass},
		"api/mail":          {Status: health.StatusWarn, Error: "slow"},
		"worker/redis":      {Status: health.StatusPass},
		"worker/queue":      {Status: health.StatusFail, Error: "down"},
		"standalone-health": {Status: health.StatusPass},
	}
}

func TestGroupChecksBySource_PartitionsAndRollsUp(t *testing.T) {
	t.Parallel()

	groups := groupChecksBySource(sourceChecks())

	if len(groups) != 3 {
		t.Fatalf("groups: want 3 (api, worker, Services), got %d: %+v", len(groups), groups)
	}

	want := []struct {
		title  string
		status health.Status
		rows   int
	}{
		{title: "Services", status: health.StatusPass, rows: 1},
		{title: "api", status: health.StatusWarn, rows: 2},
		{title: "worker", status: health.StatusFail, rows: 2},
	}

	for i, expected := range want {
		if groups[i].Title != expected.title {
			t.Errorf("group %d title: want %q, got %q", i, expected.title, groups[i].Title)
		}

		if groups[i].Status != expected.status {
			t.Errorf(
				"group %q status: want %s, got %s",
				expected.title,
				expected.status,
				groups[i].Status,
			)
		}

		if len(groups[i].Rows) != expected.rows {
			t.Errorf(
				"group %q rows: want %d, got %d",
				expected.title,
				expected.rows,
				len(groups[i].Rows),
			)
		}
	}
}

func TestGroupChecksBySource_SortsRowsByName(t *testing.T) {
	t.Parallel()

	groups := groupChecksBySource(sourceChecks())

	for _, group := range groups {
		for i := 1; i < len(group.Rows); i++ {
			if group.Rows[i-1].Name > group.Rows[i].Name {
				t.Errorf("group %q rows not sorted: %q before %q",
					group.Title, group.Rows[i-1].Name, group.Rows[i].Name)
			}
		}
	}
}

func TestGroupChecksBy_DispatchesAndFallsBack(t *testing.T) {
	t.Parallel()

	checks := sourceChecks()

	if got := len(groupChecksBy(GroupBySource, checks)); got != 3 {
		t.Errorf("GroupBySource: want 3 groups, got %d", got)
	}

	if got := len(groupChecksBy(GroupBySeverity, checks)); got != 3 {
		t.Errorf("GroupBySeverity: want 3 severity groups, got %d", got)
	}

	if got := len(groupChecksBy(GroupMode("nonsense"), checks)); got != 3 {
		t.Errorf("unknown mode must fall back to severity grouping, got %d groups", got)
	}
}

func TestBuildViewModel_GroupBySourceSetsGrouping(t *testing.T) {
	t.Parallel()

	vm := buildViewModel(health.Response{Checks: sourceChecks()}, "T", "/sse", GroupBySource)

	if vm.Grouping != GroupBySource {
		t.Errorf("Grouping: want source, got %q", vm.Grouping)
	}

	if len(vm.Groups) != 3 {
		t.Errorf("groups: want 3, got %d", len(vm.Groups))
	}
}

func TestApplyCollapsePolicy_SkipsSourceGrouping(t *testing.T) {
	t.Parallel()

	vm := buildViewModel(health.Response{Checks: sourceChecks()}, "T", "/sse", GroupBySource)
	applyCollapsePolicy(
		&vm,
		1,
	) // threshold 1 with 2-row pass sources would collapse in severity mode

	if vm.HealthyCount != 0 {
		t.Errorf("HealthyCount must stay 0 in source mode, got %d", vm.HealthyCount)
	}

	if vm.HealthyCollapsed {
		t.Error("source mode must never auto-collapse via the severity policy")
	}
}

func TestAnonymizeViewModel_MasksSourceTitles(t *testing.T) {
	t.Parallel()

	vm := buildViewModel(health.Response{Checks: sourceChecks()}, "T", "/sse", GroupBySource)
	anonymizeViewModel(&vm)

	for i, group := range vm.Groups {
		if group.Title == "api" || group.Title == "worker" || group.Title == groupTitleServices {
			t.Errorf("group %d title %q leaks the source topology in public mode", i, group.Title)
		}
	}
}

func TestStorageKey_NeverPersistsInSourceMode(t *testing.T) {
	t.Parallel()

	vm := buildViewModel(health.Response{Checks: sourceChecks()}, "T", "/sse", GroupBySource)
	vm.PersistCollapse = true

	if key := storageKey(vm); key != "" {
		t.Errorf(
			"storageKey in source mode: want empty (several pass sections must not share one key), got %q",
			key,
		)
	}

	severity := buildViewModel(
		health.Response{Checks: sourceChecks()},
		"T",
		"/sse",
		GroupBySeverity,
	)
	severity.PersistCollapse = true

	if key := storageKey(severity); key != collapseStorageKeyName {
		t.Errorf("storageKey in severity mode: want %q, got %q", collapseStorageKeyName, key)
	}
}

// TestGroupChecks_NeverEmitEmptyGroups is the zero-count property guard:
// no grouping mode may produce an empty group (the badge pluralization
// helper and the healthy summary are unreachable at 0 by construction).
// Sweeps deterministic edge maps plus pseudo-random subsets.
func TestGroupChecks_NeverEmitEmptyGroups(t *testing.T) {
	t.Parallel()

	allStatuses := []health.Status{
		health.StatusPass,
		health.StatusWarn,
		health.StatusFail,
		"unknown",
	}

	fixtures := make([]map[string]health.Check, 0, 53)
	fixtures = append(fixtures,
		map[string]health.Check{},
		map[string]health.Check{"only": {Status: health.StatusPass}},
		map[string]health.Check{
			"a": {Status: health.StatusFail},
			"b": {Status: health.StatusWarn},
			"c": {Status: health.StatusPass},
		},
	)

	for i := range 50 {
		checks := map[string]health.Check{}

		for j := range i%7 + 1 {
			checks[string(rune('a'+j%26))+string(rune('a'+i%26))] = health.Check{
				Status: allStatuses[(i+j)%len(allStatuses)],
			}
		}

		fixtures = append(fixtures, checks)
	}

	for fixtureIndex, checks := range fixtures {
		for _, mode := range []GroupMode{GroupBySeverity, GroupBySource} {
			for _, group := range groupChecksBy(mode, checks) {
				if len(group.Rows) == 0 {
					t.Errorf(
						"fixture %d mode %s: group %q has zero rows — empty groups must never render",
						fixtureIndex,
						mode,
						group.Title,
					)
				}
			}
		}
	}
}
