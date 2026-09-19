package app

import "net/http"

// The parts page (test mode only): every shared interface part of SPEC
// B9.5 on one page, drawn by the real partials in web/templates/app/
// parts.html. The design tests compare each one with docs/design/
// mockup.html, and run the accessibility and window-size checks once over
// the lot. Its sample values are hard-coded because internal/app can't
// import the section that owns the real ones (SPEC B2).

type testPartsPerson struct {
	Name       string
	Initials   string
	ColorClass string
	Removed    bool
}

type testPartsBlocks struct {
	TaskID int64
	Blocks []int
}

type testPartsLane struct {
	Workload      []testPartsBlocks
	WorkloadLine  string
	WorkloadLabel string
}

type testPartsData struct {
	People   []testPartsPerson
	Lane     testPartsLane
	Sizes    []string
	MeClass  string
	Selected string
}

func (s *Server) handleTestParts(w http.ResponseWriter, r *http.Request) {
	s.RenderFrame(w, r, http.StatusOK, "test-parts.html", "Parts", testPartsData{
		People: []testPartsPerson{
			{Name: "Priya", Initials: "PR", ColorClass: "av me"},
			{Name: "Sam", Initials: "SA", ColorClass: "av c"},
			{Name: "Jo", Initials: "JO", ColorClass: "av b"},
			{Name: "Alex", Initials: "AL", ColorClass: "av d"},
			{Name: "Kim", Initials: "KI", ColorClass: "av"},
		},
		Lane: testPartsLane{
			Workload:      []testPartsBlocks{{TaskID: 1, Blocks: make([]int, 4)}, {TaskID: 2, Blocks: make([]int, 2)}, {TaskID: 3, Blocks: make([]int, 1)}},
			WorkloadLine:  "1 large · 1 medium · 1 small",
			WorkloadLabel: "Workload: 7 blocks — large, medium, small",
		},
		Sizes: []string{"Small", "Medium", "Large"},
	})
}
