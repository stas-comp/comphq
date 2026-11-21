package tasks

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The guard behind gate 8.42: the behavioural test lists the screens that exist
// today, and this one catches the next. Any query in this package that lists jobs
// (it selects job ids and titles from tasks, and isn't a lookup of one job by id)
// has to apply waitingClause, or be named below with the reason it is allowed to
// show a waiting yearly job. A new list that forgets the clause fails here, not
// in front of the staff as a job that appears months early.
func TestEveryQueryThatListsJobsKnowsAboutWaiting(t *testing.T) {
	allowed := map[string]string{
		"due.go":               "the Calendar shows a yearly job on its due date, waiting or not (gate 8.49)",
		"search.go":            "job search and the linked-job picker find waiting jobs, marked (gate 8.50)",
		"export.go":            "Export everything lists every job ever made",
		"lifecycle.go":         "Finished tasks leaves yearly jobs out by repeat, and Removed tasks lists only removed jobs",
		"references.go":        "Jobs using this article lists real jobs, waiting or not",
		"references_export.go": "Export everything lists every article and place link",
		"links.go":             "a job's own Linked jobs and the link export list jobs the people linked on purpose",
	}
	listQuery := regexp.MustCompile("(?s)`(\\s*SELECT\\s+(?:DISTINCT\\s+)?(?:t\\.|a\\.|o\\.)?id,\\s*(?:t\\.|a\\.|o\\.)?title[^`]*?(?:FROM|JOIN)\\s+tasks[^`]*)`(\\s*\\+\\s*waitingClause)?")
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range listQuery.FindAllStringSubmatch(string(src), -1) {
			query := strings.Join(strings.Fields(m[1]), " ")
			byID := strings.Contains(query, "WHERE id = ?") || strings.Contains(query, "WHERE t.id = ?")
			if byID || m[2] != "" || strings.Contains(query, "waitingClause") {
				continue
			}
			if _, ok := allowed[f]; !ok {
				t.Errorf("%s has a query that lists jobs without waitingClause: %.120s...\n"+
					"A waiting yearly job must not appear there (gate 8.42): add `+ waitingClause(...)`, or name the file in this test with the reason.", f, query)
			}
		}
	}
}
