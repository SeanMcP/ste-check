// Package ste checks text against a subset of ASD-STE100 Simplified Technical English rules.
package ste

import (
	_ "embed"
	"fmt"
	"iter"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

type TextType string

const (
	Procedure   TextType = "procedure"
	Description TextType = "description"
)

var DefaultMaxWords = map[TextType]int{Procedure: 20, Description: 25}

type Rule string

const (
	Length      Rule = "max-sentence-length"
	Word        Rule = "no-unapproved-words"
	Ing         Rule = "no-ing"
	Passive     Rule = "no-passive"
	Contraction Rule = "no-contractions"
	Tense       Rule = "no-perfect-tense"
)

var Rules = []Rule{Length, Word, Ing, Passive, Contraction, Tense}

// Issue is one rule violation. Start and End are byte offsets in the
// sentence text. They are both 0 when the issue applies to the whole sentence.
type Issue struct {
	Rule       Rule
	Start, End int
	Message    string
}

type Sentence struct {
	Text   string
	Issues []Issue
}

//go:embed words.txt
var wordsFile string

//go:embed ing-allowed.txt
var ingAllowedFile string

// Options changes the checker. The zero value uses the defaults.
type Options struct {
	MaxWords map[TextType]int  // Replaces the default limit for each type it has.
	Disabled map[Rule]bool     // Rules to skip.
	Words    map[string]string // More unapproved words or phrases, with alternatives.
	AllowIng []string          // More -ing words to allow.
	Ignore   []string          // Words that no rule flags.
}

type Checker struct {
	maxWords   map[TextType]int
	disabled   map[Rule]bool
	unapproved map[string]string // word or phrase -> alternative
	wordForms  map[string]string // inflected form -> unapproved base word
	phrases    [][]string        // longest first
	ingAllowed map[string]bool
	ignore     [][]string // words and phrases that no rule flags
}

var defaultChecker = New(Options{})

// Check checks text with the default options.
func Check(text string, t TextType) []Sentence {
	return defaultChecker.Check(text, t)
}

func New(o Options) *Checker {
	c := &Checker{
		maxWords:   maps.Clone(DefaultMaxWords),
		disabled:   maps.Clone(o.Disabled),
		unapproved: map[string]string{},
		wordForms:  map[string]string{},
		ingAllowed: map[string]bool{},
	}
	maps.Copy(c.maxWords, o.MaxWords)

	for line := range lines(wordsFile) {
		bad, good, ok := strings.Cut(line, " = ")
		if !ok {
			panic("words.txt: bad line: " + line)
		}
		c.unapproved[bad] = good
	}
	for bad, good := range o.Words {
		c.unapproved[strings.Join(strings.Fields(strings.ToLower(bad)), " ")] = good
	}
	for base := range c.unapproved {
		if strings.Contains(base, " ") {
			c.phrases = append(c.phrases, strings.Fields(base))
			continue
		}
		for _, form := range inflections(base) {
			c.wordForms[form] = base
		}
	}
	// Longest first, so "due to the fact that" wins over "due to".
	slices.SortFunc(c.phrases, func(a, b []string) int { return len(b) - len(a) })

	for w := range lines(ingAllowedFile) {
		c.ingAllowed[w] = true
	}
	for _, w := range o.AllowIng {
		c.ingAllowed[strings.ToLower(w)] = true
	}
	for _, w := range o.Ignore {
		if words := strings.Fields(strings.ToLower(strings.ReplaceAll(w, "’", "'"))); len(words) > 0 {
			c.ignore = append(c.ignore, words)
		}
	}
	return c
}

func (c *Checker) MaxWords(t TextType) int {
	return c.maxWords[t]
}

// lines returns the lines of a data file, without blank lines and # comments.
func lines(file string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for line := range strings.SplitSeq(file, "\n") {
			if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") && !yield(line) {
				return
			}
		}
	}
}

var irregularParticiples = set(
	"been", "become", "begun", "bought", "broken", "brought", "built", "caught", "chosen",
	"come", "cut", "dealt", "done", "drawn", "driven", "eaten", "fallen", "fed", "felt",
	"flown", "forgotten", "found", "frozen", "given", "gone", "got", "gotten", "grown",
	"heard", "held", "hidden", "hung", "kept", "known", "laid", "led", "left", "lost",
	"made", "meant", "met", "paid", "put", "read", "ridden", "risen", "run", "said",
	"seen", "sent", "set", "shaken", "shot", "shown", "slept", "sold", "spent", "split",
	"spoken", "spread", "stolen", "stood", "struck", "taken", "taught", "thought",
	"thrown", "told", "torn", "understood", "woken", "won", "worn", "written",
)

// Words that match "-ed" but are not participles.
var notParticiples = set("need", "speed", "feed", "seed", "weed", "deed", "reed", "creed", "greed", "steed", "sled")

// Words that can come between "have" and a participle: "have not seen".
var perfectAdverbs = set("not", "never", "already", "just", "also", "always", "recently", "ever")

// Contractions that end in "'s" and are never possessive.
var sContractions = map[string]string{
	"it's": "it is", "that's": "that is", "there's": "there is", "here's": "here is",
	"what's": "what is", "who's": "who is", "he's": "he is", "she's": "she is", "let's": "let us",
}

var contractionSuffixes = []struct{ suffix, full string }{
	{"n't", " not"}, {"'re", " are"}, {"'ve", " have"}, {"'ll", " will"}, {"'m", " am"}, {"'d", ""},
}

var (
	wordRE = regexp.MustCompile(`(?i)[a-z0-9]+(?:[-'’][a-z0-9]+)*`)
	edRE   = regexp.MustCompile(`^[a-z]{2,}ed$`)
)

// inflections returns likely forms of a word, so "verified" matches "verify".
func inflections(w string) []string {
	forms := []string{w, w + "s", w + "es", w + "d", w + "ed", w + "ing"}
	if stem, ok := strings.CutSuffix(w, "y"); ok {
		forms = append(forms, stem+"ies", stem+"ied")
	}
	if stem, ok := strings.CutSuffix(w, "e"); ok {
		forms = append(forms, stem+"ing")
	}
	return forms
}

type token struct {
	word       string // lowercase
	start, end int
}

func tokenize(s string) []token {
	var tokens []token
	for _, loc := range wordRE.FindAllStringIndex(s, -1) {
		w := strings.ToLower(strings.ReplaceAll(s[loc[0]:loc[1]], "’", "'"))
		tokens = append(tokens, token{w, loc[0], loc[1]})
	}
	return tokens
}

// SplitSentences splits on end punctuation followed by a space, and on line breaks.
func SplitSentences(text string) []string {
	var out []string
	add := func(s string) {
		if s = strings.TrimSpace(s); wordRE.MatchString(s) {
			out = append(out, s)
		}
	}
	for line := range strings.SplitSeq(text, "\n") {
		start := 0
		for i := 0; i+1 < len(line); i++ {
			if strings.IndexByte(".!?", line[i]) >= 0 && (line[i+1] == ' ' || line[i+1] == '\t') {
				add(line[start : i+1])
				start = i + 1
			}
		}
		add(line[start:])
	}
	return out
}

func (c *Checker) Check(text string, t TextType) []Sentence {
	var sentences []Sentence
	for _, s := range SplitSentences(text) {
		sentences = append(sentences, Sentence{Text: s, Issues: c.checkSentence(s, t)})
	}
	return sentences
}

func (c *Checker) checkSentence(s string, t TextType) []Issue {
	tokens := tokenize(s)
	skip := make([]bool, len(tokens)) // tokens inside an ignored word or phrase
	for _, phrase := range c.ignore {
		for i := 0; i+len(phrase) <= len(tokens); i++ {
			if matches(tokens[i:i+len(phrase)], phrase) {
				for j := i; j < i+len(phrase); j++ {
					skip[j] = true
				}
			}
		}
	}
	ignored := func(i int) bool { return skip[i] }
	on := func(r Rule) bool { return !c.disabled[r] }
	var issues []Issue

	if n, max := len(tokens), c.maxWords[t]; on(Length) && n > max {
		issues = append(issues, Issue{Rule: Length,
			Message: fmt.Sprintf("%d words. The limit for a %s is %d.", n, t, max)})
	}

	covered := make([]bool, len(tokens)) // tokens already inside a flagged phrase
	for _, phrase := range c.phrases {
		if !on(Word) {
			break
		}
		n := len(phrase)
		for i := 0; i+n <= len(tokens); i++ {
			if !matches(tokens[i:i+n], phrase) || slices.Contains(covered[i:i+n], true) ||
				slices.Contains(skip[i:i+n], true) {
				continue
			}
			for j := i; j < i+n; j++ {
				covered[j] = true
			}
			issues = append(issues, c.unapprovedIssue(s, tokens[i].start, tokens[i+n-1].end, strings.Join(phrase, " ")))
		}
	}

	for i, tok := range tokens {
		if ignored(i) {
			continue
		}
		text := s[tok.start:tok.end]
		if full, ok := expandContraction(text); ok && on(Contraction) {
			issues = append(issues, contraction(text, full, tok))
		}
		if covered[i] {
			continue
		}
		if base, ok := c.wordForms[tok.word]; ok && on(Word) {
			issues = append(issues, c.unapprovedIssue(s, tok.start, tok.end, base))
		} else if !ok && on(Ing) && c.isIngVerb(tok, s, i) {
			issues = append(issues, Issue{Ing, tok.start, tok.end,
				fmt.Sprintf("%q is an -ing form. Use a different verb form.", text)})
		}
	}

	for i := 0; on(Passive) && i+1 < len(tokens); i++ {
		if isPassive(tokens, i) && !ignored(i) && !ignored(i+1) {
			aux, part := tokens[i], tokens[i+1]
			issues = append(issues, Issue{Passive, aux.start, part.end,
				fmt.Sprintf("%q is passive voice. Say who does the action.", s[aux.start:part.end])})
		}
	}

	for i := 0; on(Tense) && i < len(tokens); i++ {
		if j, ok := perfectTense(tokens, i); ok && !ignored(i) && !ignored(j) {
			start, end := tokens[i].start, tokens[j].end
			issues = append(issues, Issue{Tense, start, end,
				fmt.Sprintf("%q is a perfect tense. Use the simple past or present.", s[start:end])})
		}
	}

	slices.SortStableFunc(issues, func(a, b Issue) int { return a.Start - b.Start })
	return issues
}

// isIngVerb reports whether a token is an -ing verb form. A capitalized word
// after the first word is a proper noun, such as "Thanksgiving" or "Beijing".
func (c *Checker) isIngVerb(tok token, s string, i int) bool {
	properNoun := i > 0 && unicode.IsUpper(rune(s[tok.start]))
	last := tok.word[strings.LastIndexByte(tok.word, '-')+1:] // "no-ing" -> "ing"
	return len(last) > 4 && strings.HasSuffix(last, "ing") && !c.ingAllowed[tok.word] && !properNoun
}

func (c *Checker) unapprovedIssue(s string, start, end int, base string) Issue {
	return Issue{Word, start, end,
		fmt.Sprintf("Use %q, not %q.", c.unapproved[base], s[start:end])}
}

func matches(tokens []token, phrase []string) bool {
	for i, w := range phrase {
		if tokens[i].word != w {
			return false
		}
	}
	return true
}

func contraction(text, full string, tok token) Issue {
	msg := fmt.Sprintf("%q is a contraction. Write %q.", text, full)
	if full == "" {
		msg = fmt.Sprintf("%q is a contraction. Write the full words.", text)
	}
	return Issue{Contraction, tok.start, tok.end, msg}
}

// expandContraction returns the full form of a contraction, keeping the first
// letter's case. The full form is "" when it is ambiguous ("I'd" can be "I had" or "I would").
func expandContraction(text string) (string, bool) {
	text = strings.ReplaceAll(text, "’", "'")
	lower := strings.ToLower(text)
	full, ok := map[string]string{"can't": "cannot", "won't": "will not", "shan't": "shall not"}[lower]
	if !ok {
		full, ok = sContractions[lower]
	}
	if !ok {
		for _, c := range contractionSuffixes {
			if strings.HasSuffix(lower, c.suffix) {
				if c.full == "" {
					return "", true
				}
				full, ok = lower[:len(lower)-len(c.suffix)]+c.full, true
				break
			}
		}
	}
	if !ok {
		return "", false
	}
	if unicode.IsUpper(rune(text[0])) {
		full = strings.ToUpper(full[:1]) + full[1:]
	}
	return full, true
}

// perfectTense reports whether a perfect tense ("have seen", "hasn't been")
// starts at tokens[i], and returns the index of the participle.
func perfectTense(tokens []token, i int) (int, bool) {
	switch aux := tokens[i].word; {
	case aux == "have", aux == "has", aux == "had", aux == "haven't", aux == "hasn't",
		aux == "hadn't", strings.HasSuffix(aux, "'ve"):
	default:
		return 0, false
	}
	j := i + 1
	for j < len(tokens) && perfectAdverbs[tokens[j].word] {
		j++
	}
	if j < len(tokens) && isParticiple(tokens[j].word) {
		return j, true
	}
	return 0, false
}

func isParticiple(w string) bool {
	return (edRE.MatchString(w) && !notParticiples[w]) || irregularParticiples[w]
}

// isPassive reports whether tokens[i] and tokens[i+1] form a passive verb.
// "must be removed" is always passive. "is closed" can be a state, so it only
// counts as passive when an agent follows ("is closed by the pilot").
func isPassive(tokens []token, i int) bool {
	aux, next := tokens[i].word, tokens[i+1].word
	if !isParticiple(next) {
		return false
	}
	switch aux {
	case "be", "been", "being":
		return true
	case "is", "are", "was", "were":
		return slices.ContainsFunc(tokens[i+2:], func(t token) bool { return t.word == "by" })
	}
	return false
}

func set(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}
