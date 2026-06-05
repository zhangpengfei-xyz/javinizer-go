package matcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectPartSuffix(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		wantNum     int
		wantSuf     string
		wantPattern string
	}{
		// ── Explicit: pt/part with dash separator ──────────────────────
		{"IPX-535-pt1", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535-pt2", "IPX-535", 2, "-pt2", PatternExplicit},
		{"IPX-535-pt10", "IPX-535", 10, "-pt10", PatternExplicit},
		{"IPX-535-pt99", "IPX-535", 99, "-pt99", PatternExplicit},
		{"IPX-535-part1", "IPX-535", 1, "-part1", PatternExplicit},
		{"IPX-535-part2", "IPX-535", 2, "-part2", PatternExplicit},

		// ── Explicit: pt/part without separator (noisy concatenation) ─
		{"IPX-535PT2", "IPX-535", 2, "-pt2", PatternExplicit},
		{"IPX-535part2", "IPX-535", 2, "-part2", PatternExplicit},

		// ── Explicit: pt/part with space separator ────────────────────
		{"IPX-535 pt1", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535 part2", "IPX-535", 2, "-part2", PatternExplicit},

		// ── Explicit: pt/part with underscore separator ───────────────
		{"IPX-535_pt1", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535_part3", "IPX-535", 3, "-part3", PatternExplicit},

		// ── Explicit: pt/part with dot separator ──────────────────────
		{"IPX-535.pt1", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535.part1", "IPX-535", 1, "-part1", PatternExplicit},

		// ── Explicit: plain number is entire remainder ────────────────
		{"PRED-151-1", "PRED-151", 1, "-1", PatternExplicit},
		{"PRED-151-2", "PRED-151", 2, "-2", PatternExplicit},
		{"IPX-535_1", "IPX-535", 1, "-1", PatternExplicit},
		{"IPX-535.1", "IPX-535", 1, "-1", PatternExplicit},
		{"IPX-535 2", "IPX-535", 2, "-2", PatternExplicit},
		{"IPX-5351", "IPX-535", 1, "-1", PatternExplicit}, // no separator, bare digit

		// ── Explicit: pt/part with text after ────────────────────────
		{"IPX-535-pt1-extra", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535-part1 Disc1", "IPX-535", 1, "-part1", PatternExplicit},

		// ── Ambiguous: letter suffixes ────────────────────────────────
		{"MDB-087A", "MDB-087", 1, "-A", PatternLetter},
		{"MDB-087-b", "MDB-087", 2, "-B", PatternLetter},
		{"ABP-123c", "ABP-123", 3, "-C", PatternLetter},
		{"IPX-535-D", "IPX-535", 4, "-D", PatternLetter},
		{"IPX-535-Z", "IPX-535", 26, "-Z", PatternLetter},
		{"ABW-121-C", "ABW-121", 3, "-C", PatternLetter},
		{"IPX-535.A", "IPX-535", 1, "-A", PatternLetter},
		{"IPX-535_A", "IPX-535", 1, "-A", PatternLetter},

		// ── Trailing: number after noise (site tags) ──────────────────
		{"SGKI-071-un-javgg.net-1", "SGKI-071", 1, "-1", PatternTrailing},
		{"SGKI-071-un-javgg.net-2", "SGKI-071", 2, "-2", PatternTrailing},
		{"IPX-535-HD-1", "IPX-535", 1, "-1", PatternTrailing},
		{"IPX-535-HD-2", "IPX-535", 2, "-2", PatternTrailing},
		{"ABC-123_site.name_1", "ABC-123", 1, "-1", PatternTrailing},
		{"IPX-535.javdb.1", "IPX-535", 1, "-1", PatternTrailing},
		{"IPX-535-uncen-1", "IPX-535", 1, "-1", PatternTrailing},
		{"IPX-535-leak-2", "IPX-535", 2, "-2", PatternTrailing},
		{"SSIS-001-h_1245svhr003-1", "SSIS-001", 1, "-1", PatternTrailing},

		// ── No pattern ───────────────────────────────────────────────
		{"ABC-123", "ABC-123", 0, "", PatternNone},
		{"IPX-535 no suffix", "IPX-535", 0, "", PatternNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, suf, pattern, _ := DetectPartSuffix(tt.name, tt.id)
			assert.Equal(t, tt.wantNum, num, "PartNumber mismatch")
			assert.Equal(t, tt.wantSuf, suf, "PartSuffix mismatch")
			assert.Equal(t, tt.wantPattern, pattern, "PatternType mismatch")
		})
	}
}

func TestDetectPartSuffix_PatternConstants(t *testing.T) {
	assert.Equal(t, "explicit", PatternExplicit)
	assert.Equal(t, "letter", PatternLetter)
	assert.Equal(t, "trailing", PatternTrailing)
	assert.Equal(t, "", PatternNone)
}

// TestDetectPartSuffix_FalseNegatives verifies patterns that should NOT be detected
// as multipart, preventing unintended file rename side effects.
func TestDetectPartSuffix_FalseNegatives(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		wantNum     int
		wantSuf     string
		wantPattern string
	}{
		// ── Resolution / codec tags ───────────────────────────────────
		{"IPX-535-1080p", "IPX-535", 0, "", PatternNone},
		{"IPX-535-720p", "IPX-535", 0, "", PatternNone},
		{"IPX-535-4K", "IPX-535", 0, "", PatternNone},
		{"IPX-535.FHD", "IPX-535", 0, "", PatternNone},
		{"IPX-535.1080p", "IPX-535", 0, "", PatternNone},
		{"IPX-535.720p", "IPX-535", 0, "", PatternNone},

		// ── Version / edition tags ────────────────────────────────────
		{"IPX-535-v2", "IPX-535", 0, "", PatternNone},
		{"IPX-535-v10", "IPX-535", 0, "", PatternNone},
		{"IPX-535-r1", "IPX-535", 0, "", PatternNone},

		// ── cd1 / cd2 (letter before digit, not separator+digit) ─────
		{"IPX-535-cd1", "IPX-535", 0, "", PatternNone},
		{"IPX-535-cd2", "IPX-535", 0, "", PatternNone},

		// ── Years / dates (4+ digits not matched by \d{1,2}) ────────
		{"IPX-535-2020", "IPX-535", 0, "", PatternNone},
		{"IPX-535.2024", "IPX-535", 0, "", PatternNone},
		{"IPX-535-20200315", "IPX-535", 0, "", PatternNone},

		// ── ID is the entire name (no remainder) ─────────────────────
		{"IPX-535", "IPX-535", 0, "", PatternNone},

		// ── Title text ───────────────────────────────────────────────
		{"IPX-535 Beautiful Day", "IPX-535", 0, "", PatternNone},
		{"IPX-535 Title With Many Words", "IPX-535", 0, "", PatternNone},

		// ── E/Z suffix is part of the ID, not a part indicator ──────
		// (These IDs include E/Z in the match, so remainder is empty)
		{"IPX-535E", "IPX-535E", 0, "", PatternNone},
		{"IPX-535Z", "IPX-535Z", 0, "", PatternNone},

		// ── pt0 / part0 are not valid part numbers ──────────────────
		{"IPX-535-pt0", "IPX-535", 0, "", PatternNone},
		{"IPX-535-part0", "IPX-535", 0, "", PatternNone},

		// ── pt100 / part100 exceed 2-digit limit ─────────────────────
		{"IPX-535-pt100", "IPX-535", 0, "", PatternNone},
		{"IPX-535-part100", "IPX-535", 0, "", PatternNone},

		// ── Letter with text after prevents letter detection ─────────
		{"IPX-535-A-extra", "IPX-535", 0, "", PatternNone},
		{"IPX-535-A 1080p", "IPX-535", 0, "", PatternNone},

		// ── Multi-letter remainder is not a letter part ──────────────
		{"IPX-535-AB", "IPX-535", 0, "", PatternNone},
		{"IPX-535-XX", "IPX-535", 0, "", PatternNone},

		// ── Trailing -0 is not a valid part number ──────────────────
		{"IPX-535-HD-0", "IPX-535", 0, "", PatternNone},

		// ── Trailing 3-digit number exceeds limit ───────────────────
		{"IPX-535-HD-100", "IPX-535", 0, "", PatternNone},

		// ── Letter+number: step 3 (trailing) preempts step 4 (letter) ─
		{"IPX-535-A-1", "IPX-535", 1, "-1", PatternTrailing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, suf, pattern, _ := DetectPartSuffix(tt.name, tt.id)
			assert.Equal(t, tt.wantNum, num, "PartNumber mismatch")
			assert.Equal(t, tt.wantSuf, suf, "PartSuffix mismatch")
			assert.Equal(t, tt.wantPattern, pattern, "PatternType mismatch")
		})
	}
}

// TestDetectPartSuffix_CaseInsensitivity verifies that pt/part/letter detection
// works regardless of case.
func TestDetectPartSuffix_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		wantNum     int
		wantSuf     string
		wantPattern string
	}{
		// pt / PT / Pt / pT
		{"IPX-535-pt1", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535-PT1", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535-Pt1", "IPX-535", 1, "-pt1", PatternExplicit},

		// part / PART / Part
		{"IPX-535-part1", "IPX-535", 1, "-part1", PatternExplicit},
		{"IPX-535-PART1", "IPX-535", 1, "-part1", PatternExplicit},
		{"IPX-535-Part1", "IPX-535", 1, "-part1", PatternExplicit},

		// Letter suffixes normalized to uppercase
		{"IPX-535-a", "IPX-535", 1, "-A", PatternLetter},
		{"IPX-535-A", "IPX-535", 1, "-A", PatternLetter},
		{"IPX-535-z", "IPX-535", 26, "-Z", PatternLetter},
		{"IPX-535-Z", "IPX-535", 26, "-Z", PatternLetter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, suf, pattern, _ := DetectPartSuffix(tt.name, tt.id)
			assert.Equal(t, tt.wantNum, num, "PartNumber mismatch")
			assert.Equal(t, tt.wantSuf, suf, "PartSuffix mismatch")
			assert.Equal(t, tt.wantPattern, pattern, "PatternType mismatch")
		})
	}
}

// TestDetectPartSuffix_RealWorldFilenames tests multipart detection against
// real-world filename patterns encountered in the wild.
func TestDetectPartSuffix_RealWorldFilenames(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		wantNum     int
		wantSuf     string
		wantPattern string
	}{
		// Downloaded files with site tags
		{"SGKI-071-un-javgg.net-1", "SGKI-071", 1, "-1", PatternTrailing},
		{"SGKI-071-un-javgg.net-2", "SGKI-071", 2, "-2", PatternTrailing},
		{"SSIS-001-HD-1", "SSIS-001", 1, "-1", PatternTrailing},
		{"SSIS-001-HD-2", "SSIS-001", 2, "-2", PatternTrailing},

		// Bracket-wrapped site tags
		{"IPX-535]-[ThZu.Cc]-1", "IPX-535", 1, "-1", PatternTrailing},

		// Dot-separated from download sites
		{"IPX-535.javdb.1", "IPX-535", 1, "-1", PatternTrailing},
		{"ABC-123.javbus.2", "ABC-123", 2, "-2", PatternTrailing},

		// Standard clean multi-part
		{"IPX-535-pt1", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535-pt2", "IPX-535", 2, "-pt2", PatternExplicit},
		{"PRED-151-1", "PRED-151", 1, "-1", PatternExplicit},
		{"PRED-151-2", "PRED-151", 2, "-2", PatternExplicit},

		// Letter-based parts
		{"MDB-087-A", "MDB-087", 1, "-A", PatternLetter},
		{"MDB-087-B", "MDB-087", 2, "-B", PatternLetter},

		// Single file that looks like it could be multi but isn't (no sibling)
		{"ABW-121-C", "ABW-121", 3, "-C", PatternLetter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, suf, pattern, _ := DetectPartSuffix(tt.name, tt.id)
			assert.Equal(t, tt.wantNum, num, "PartNumber mismatch")
			assert.Equal(t, tt.wantSuf, suf, "PartSuffix mismatch")
			assert.Equal(t, tt.wantPattern, pattern, "PatternType mismatch")
		})
	}
}

// TestDetectPartSuffix_BoundaryValues tests numeric boundaries.
func TestDetectPartSuffix_BoundaryValues(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		wantNum     int
		wantSuf     string
		wantPattern string
	}{
		// Part 0 is not valid
		{"IPX-535-pt0", "IPX-535", 0, "", PatternNone},
		{"IPX-535-part0", "IPX-535", 0, "", PatternNone},
		{"IPX-535-0", "IPX-535", 0, "", PatternNone},

		// Part 1 (minimum valid)
		{"IPX-535-pt1", "IPX-535", 1, "-pt1", PatternExplicit},
		{"IPX-535-1", "IPX-535", 1, "-1", PatternExplicit},

		// Part 99 (2-digit max)
		{"IPX-535-pt99", "IPX-535", 99, "-pt99", PatternExplicit},
		{"IPX-535-99", "IPX-535", 99, "-99", PatternExplicit},

		// Part 100 exceeds 2-digit limit
		{"IPX-535-pt100", "IPX-535", 0, "", PatternNone},
		{"IPX-535-100", "IPX-535", 0, "", PatternNone},

		// Letter A = part 1, Z = part 26
		{"IPX-535-A", "IPX-535", 1, "-A", PatternLetter},
		{"IPX-535-Z", "IPX-535", 26, "-Z", PatternLetter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, suf, pattern, _ := DetectPartSuffix(tt.name, tt.id)
			assert.Equal(t, tt.wantNum, num, "PartNumber mismatch")
			assert.Equal(t, tt.wantSuf, suf, "PartSuffix mismatch")
			assert.Equal(t, tt.wantPattern, pattern, "PatternType mismatch")
		})
	}
}

// TestDetectPartSuffix_SuffixNormalization verifies that the suffix string
// is always lowercase and dash-prefixed regardless of input case.
func TestDetectPartSuffix_SuffixNormalization(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantSuf string
	}{
		{"IPX-535-PT1", "IPX-535", "-pt1"},
		{"IPX-535-PART2", "IPX-535", "-part2"},
		{"IPX-535-a", "IPX-535", "-A"},
		{"IPX-535-z", "IPX-535", "-Z"},
		{"IPX-535-B", "IPX-535", "-B"},
		{"IPX-535-1", "IPX-535", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, suf, _, _ := DetectPartSuffix(tt.name, tt.id)
			assert.Equal(t, tt.wantSuf, suf)
		})
	}
}

// TestDetectPartSuffix_TrailingPrefix verifies the 4th return value — the noise
// portion before the part number for PatternTrailing results.
func TestDetectPartSuffix_TrailingPrefix(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		wantNum     int
		wantSuf     string
		wantPattern string
		wantPrefix  string
	}{
		{"SGKI-071-un-javgg.net-1", "SGKI-071", 1, "-1", PatternTrailing, "-un-javgg.net"},
		{"SGKI-071-un-javgg.net-2", "SGKI-071", 2, "-2", PatternTrailing, "-un-javgg.net"},
		{"IPX-535-HD-1", "IPX-535", 1, "-1", PatternTrailing, "-HD"},
		{"IPX-535-HD-12", "IPX-535", 12, "-12", PatternTrailing, "-HD"},
		{"IPX-535-uncen-1", "IPX-535", 1, "-1", PatternTrailing, "-uncen"},
		{"IPX-535-leak-2", "IPX-535", 2, "-2", PatternTrailing, "-leak"},
		{"IPX-535.javdb.1", "IPX-535", 1, "-1", PatternTrailing, ".javdb"},
		{"ABC-123_site.name_1", "ABC-123", 1, "-1", PatternTrailing, "_site.name"},

		// Empty prefix when remainder is just separator+digit (step 2 catches this first)
		// But if step 2 didn't fire, step 3 would give empty prefix
		{"IPX-535.1", "IPX-535", 1, "-1", PatternExplicit, ""},   // Step 2 fires first
		{"PRED-151-1", "PRED-151", 1, "-1", PatternExplicit, ""}, // Step 2 fires first

		// Explicit and letter patterns always have empty prefix
		{"IPX-535-pt1", "IPX-535", 1, "-pt1", PatternExplicit, ""},
		{"IPX-535-A", "IPX-535", 1, "-A", PatternLetter, ""},
		{"IPX-535", "IPX-535", 0, "", PatternNone, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, suf, pattern, prefix := DetectPartSuffix(tt.name, tt.id)
			assert.Equal(t, tt.wantNum, num, "PartNumber mismatch")
			assert.Equal(t, tt.wantSuf, suf, "PartSuffix mismatch")
			assert.Equal(t, tt.wantPattern, pattern, "PatternType mismatch")
			assert.Equal(t, tt.wantPrefix, prefix, "TrailingPrefix mismatch")
		})
	}
}

// TestDetectPartSuffix_EdgeCases covers unusual inputs and boundary conditions.
func TestDetectPartSuffix_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		id          string
		wantNum     int
		wantSuf     string
		wantPattern string
		wantPrefix  string
	}{
		// ID not found in filename — entire name becomes remainder
		{"ID not found", "ABC-999-pt1", "IPX-535", 1, "-pt1", PatternExplicit, ""},

		// ID is a prefix of a longer ID
		{"ID is prefix of longer", "IPX-5350-1", "IPX-535", 1, "-1", PatternTrailing, "0"},

		// ID appears multiple times — first occurrence used
		{"ID appears twice", "IPX-535-IPX-535-pt1", "IPX-535", 1, "-pt1", PatternExplicit, ""},

		// Empty inputs
		{"Empty input", "", "IPX-535", 0, "", PatternNone, ""},
		{"Empty ID", "IPX-535-pt1", "", 1, "-pt1", PatternExplicit, ""}, // entire name is remainder
		{"Both empty", "", "", 0, "", PatternNone, ""},

		// Consecutive separators
		{"Double dash", "IPX-535--1", "IPX-535", 1, "-1", PatternTrailing, "-"},
		{"Double dot", "IPX-535..2", "IPX-535", 2, "-2", PatternTrailing, "."},
		{"Double underscore", "IPX-535__3", "IPX-535", 3, "-3", PatternTrailing, "_"},

		// Non-ASCII in remainder
		{"CJK in noise", "IPX-535-美しい-1", "IPX-535", 1, "-1", PatternTrailing, "-美しい"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, suf, pattern, prefix := DetectPartSuffix(tt.input, tt.id)
			assert.Equal(t, tt.wantNum, num, "PartNumber")
			assert.Equal(t, tt.wantSuf, suf, "PartSuffix")
			assert.Equal(t, tt.wantPattern, pattern, "PatternType")
			assert.Equal(t, tt.wantPrefix, prefix, "TrailingPrefix")
		})
	}
}
