package api

import "strings"

// csvDangerous lists the leading characters that make a spreadsheet treat a
// cell as a formula: = + - @ TAB CR.
const csvDangerous = "=+-@\t\r"

// CSVSafe neutralises spreadsheet formula injection (CWE-1236, FR-BB54 AC-8,
// ISS-191) for one user-controlled TEXT cell before it is written to a CSV file.
//
// Rule: if the cell begins with '=', '+', '-', '@', TAB or CR, a single quote
// (') is prepended so Excel / LibreOffice / Sheets treat it as literal text.
// A cell that already begins with a quote followed by one of those characters
// is escaped as well, so CSVUnsafe is an exact inverse. Every other value
// (including the empty string and Unicode text) is returned unchanged. Leading
// whitespace before a formula (" =1+1") is not touched: spreadsheets do not
// evaluate it.
//
// Apply it ONLY to free-text / user-controlled fields (names, emails,
// department, titles, action / entity strings, stems, tags, JSON blobs ...).
// Do NOT apply it to numeric cells that the server formats itself (score_pct,
// time_taken_seconds, question scores, ...): a legitimate negative number such
// as "-1" must stay numeric. Server-produced timestamps and booleans never
// start with a dangerous character and need no guard either.
func CSVSafe(s string) string {
	if s == "" {
		return s
	}
	if strings.ContainsRune(csvDangerous, rune(s[0])) {
		return "'" + s
	}
	if s[0] == '\'' && len(s) >= 2 && strings.ContainsRune(csvDangerous, rune(s[1])) {
		return "'" + s
	}
	return s
}

// CSVSafeRecord returns a copy of rec with CSVSafe applied to every cell.
// Use it only for rows that consist solely of text cells.
func CSVSafeRecord(rec []string) []string {
	out := make([]string, len(rec))
	for i, c := range rec {
		out[i] = CSVSafe(c)
	}
	return out
}

// CSVUnsafe reverses CSVSafe on import so an exported-then-reimported file
// round-trips: a cell of the form '<dangerous char>... loses its leading quote.
// Cells that merely start with a quote followed by an ordinary character are
// left untouched.
func CSVUnsafe(s string) string {
	if len(s) >= 2 && s[0] == '\'' && strings.ContainsRune(csvDangerous, rune(s[1])) {
		return s[1:]
	}
	return s
}
