package ste

import (
	"slices"
	"strings"
	"testing"
)

func rules(text string, t TextType) []Rule {
	var out []Rule
	for _, s := range Check(text, t) {
		for _, issue := range s.Issues {
			out = append(out, issue.Rule)
		}
	}
	return out
}

func flagged(text string) []string {
	var out []string
	for _, s := range Check(text, Procedure) {
		for _, issue := range s.Issues {
			out = append(out, s.Text[issue.Start:issue.End])
		}
	}
	return out
}

func TestCheck(t *testing.T) {
	long := strings.Repeat("word ", 22) + "."
	cases := []struct {
		name string
		text string
		typ  TextType
		want []Rule
	}{
		{"clean instruction", "Close the valve. Make sure that the light is on.", Procedure, nil},
		{"procedure length", long, Procedure, []Rule{Length}},
		{"description length", long, Description, nil},
		{"technical -ing names", "Examine the bearing housing for damage.", Procedure, nil},
		{"-ing verb", "Continue turning the knob.", Procedure, []Rule{Ing}},
		{"modal passive", "The panel must be removed.", Procedure, []Rule{Passive}},
		{"passive with agent", "The valve is opened by the pilot.", Procedure, []Rule{Passive}},
		{"state is not passive", "Make sure that the valve is closed.", Procedure, nil},
		{"hyphenated word", "Turn off the no-ing rule.", Procedure, nil},
		{"hyphenated -ing verb", "Use self-tightening nuts.", Procedure, []Rule{Ing}},
		{"proper noun", "Return after Thanksgiving break.", Procedure, nil},
		{"-ing at sentence start", "Turning the knob opens the valve.", Procedure, []Rule{Ing}},
		{"allowed proper noun at sentence start", "Boeing builds the aircraft.", Procedure, nil},
		{"software nouns", "Add padding below the heading.", Procedure, nil},
		{"contraction", "Don't open the valve.", Procedure, []Rule{Contraction}},
		{"curly contraction", "It’s open.", Description, []Rule{Contraction}},
		{"possessive is not a contraction", "Replace the pump's seal.", Procedure, nil},
		{"perfect tense", "We have removed the cover.", Description, []Rule{Tense}},
		{"perfect tense with not", "We have not seen the fault.", Description, []Rule{Tense}},
		{"contraction and perfect tense", "I haven't heard back.", Description, []Rule{Contraction, Tense}},
		{"has with a noun", "The motor has speed control.", Description, nil},
		{"unapproved -ing is one issue", "Stop commencing the test.", Procedure, []Rule{Word}},
	}
	for _, c := range cases {
		if got := rules(c.text, c.typ); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestUnapprovedWords(t *testing.T) {
	got := flagged("Commence the test in order to verify the pressure. It was verified.")
	want := []string{"Commence", "in order to", "verify", "verified"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences(t *testing.T) {
	got := SplitSentences("Remove the cover\nOpen the door. Is it open?  Yes!\n\n")
	want := []string{"Remove the cover", "Open the door.", "Is it open?", "Yes!"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOverlappingPhrasesGiveOneIssue(t *testing.T) {
	got := flagged("It failed due to the fact that the valve was open.")
	if want := []string{"due to the fact that"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAlternativesAreNotUnapproved(t *testing.T) {
	for bad, good := range defaultChecker.unapproved {
		if issues := defaultChecker.checkSentence(good, Description); len(issues) > 0 {
			t.Errorf("%q = %q: alternative fails: %s", bad, good, issues[0].Message)
		}
	}
}

func TestIngAllowedFile(t *testing.T) {
	for w := range lines(ingAllowedFile) {
		if w != strings.ToLower(w) || !strings.HasSuffix(w, "ing") {
			t.Errorf("ing-allowed.txt: %q must be lowercase and end in -ing", w)
		}
	}
}

func TestContractionMessage(t *testing.T) {
	issues := Check("Haven't you read it? I'd go.", Description)
	got := []string{issues[0].Issues[0].Message, issues[1].Issues[0].Message}
	want := []string{`"Haven't" is a contraction. Write "Have not".`, `"I'd" is a contraction. Write the full words.`}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOptions(t *testing.T) {
	long := strings.Repeat("word ", 12) + "."
	cases := []struct {
		name string
		opts Options
		text string
		want []Rule
	}{
		{"disabled rule", Options{Disabled: map[Rule]bool{Contraction: true}}, "Don't open it.", nil},
		{"other rules still run", Options{Disabled: map[Rule]bool{Contraction: true}}, "Don't commence.", []Rule{Word}},
		{"max words", Options{MaxWords: map[TextType]int{Procedure: 10}}, long, []Rule{Length}},
		{"extra word", Options{Words: map[string]string{"Leverage": "use"}}, "Leverage the tool.", []Rule{Word}},
		{"extra phrase", Options{Words: map[string]string{"spin up": "start"}}, "Spin up the server.", []Rule{Word}},
		{"allowed -ing", Options{AllowIng: []string{"Caching"}}, "Turn on caching.", nil},
		{"ignored word", Options{Ignore: []string{"Commence"}}, "Commence the test.", nil},
		{"ignored word in phrase", Options{Ignore: []string{"prior"}}, "Prior to the test, stop.", nil},
		{"ignored phrase", Options{Ignore: []string{"Kick off"}}, "Kick off the project.", nil},
		{"ignored phrase only where it matches", Options{Ignore: []string{"kick off"}}, "Kick the tires to commence.", []Rule{Word}},
	}
	for _, c := range cases {
		var got []Rule
		for _, s := range New(c.opts).Check(c.text, Procedure) {
			for _, issue := range s.Issues {
				got = append(got, issue.Rule)
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
