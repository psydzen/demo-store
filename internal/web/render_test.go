package web

import "testing"

// TestTemplatesParse catches broken templates at test time rather than on the
// first request that renders them.
func TestTemplatesParse(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer() error = %v", err)
	}

	want := []string{
		"login", "quizzes", "play", "results", "leaderboard",
		"admin_players", "admin_quizzes", "admin_quiz", "admin_review",
	}
	for _, name := range want {
		if _, ok := r.pages[name]; !ok {
			t.Errorf("page %q was not parsed", name)
		}
	}

	for _, name := range []string{"question", "review_list", "quiz_structure"} {
		if r.fragments.Lookup(name) == nil {
			t.Errorf("fragment %q was not parsed", name)
		}
	}
}
