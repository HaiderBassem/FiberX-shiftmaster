package assistant

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Knowledge retrieval.
//
// The split of labour here is the same one the whole assistant is built on:
// the model reads the question, the database finds the documents. The model
// hands over a natural question — often an Iraqi one, often misspelled — and
// this file turns it into a ranked PostgreSQL query. It does NOT try to
// understand the question; it tokenises it, drops words that carry no signal,
// and lets ts_rank_cd decide which documents match best.
//
// Everything is local. No embedding service, no vector database, no document
// ever sent anywhere. That is not a limitation being worked around: at this
// corpus size, weighted term ranking answers the questions people ask, and it
// is auditable in a way an embedding index is not.
//
// Authorization is applied inside the SQL, exactly as the list endpoints apply
// it. A document the caller may not see is not ranked lower — it is not in the
// result set at all.

// stopWords are terms that match everything and therefore rank nothing. Both
// languages are covered because people ask in both, often in one sentence.
//
// This list is not intent detection and never decides what the user meant: it
// only stops "how do I ..." from matching every document that contains "how".
var stopWords = map[string]bool{
	// English
	"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
	"do": true, "does": true, "did": true, "how": true, "what": true, "when": true,
	"where": true, "who": true, "why": true, "can": true, "i": true, "my": true,
	"me": true, "we": true, "our": true, "you": true, "your": true, "to": true,
	"of": true, "in": true, "on": true, "for": true, "and": true, "or": true,
	"if": true, "it": true, "this": true, "that": true, "with": true, "about": true,
	"please": true, "tell": true, "show": true, "get": true, "need": true, "want": true,
	// Arabic, including the Iraqi question words
	"شنو": true, "شلون": true, "منو": true, "وين": true, "ليش": true, "شكد": true,
	"كيف": true, "ماذا": true, "متى": true, "أين": true, "لماذا": true, "هل": true,
	"من": true, "في": true, "على": true, "عن": true, "الى": true, "إلى": true,
	"مع": true, "هذا": true, "هذه": true, "ذلك": true, "التي": true, "الذي": true,
	"اذا": true, "إذا": true, "عندي": true, "عندك": true, "عندنا": true, "اكو": true,
	"اريد": true, "أريد": true, "ابي": true, "أبي": true, "كل": true, "شي": true,
	"يعني": true, "بس": true, "هسه": true, "لو": true, "او": true, "أو": true,
	"و": true, "ال": true,
}

// searchTerms reduces a question to the words worth ranking on.
//
// Splitting is by Unicode category rather than by an alphabet, so Arabic,
// English and a sentence containing both all tokenise correctly, and so does
// "الـtasks". Arabic diacritics and the tatweel are stripped, and the alef
// variants are folded, because people type them inconsistently and a document
// written with إ should be found by a question typed with ا.
func searchTerms(query string) []string {
	var b strings.Builder
	for _, r := range query {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(foldArabic(r))
		default:
			b.WriteRune(' ')
		}
	}
	seen := map[string]bool{}
	var terms []string
	for _, word := range strings.Fields(b.String()) {
		word = strings.ToLower(word)
		if len([]rune(word)) < 2 || stopWords[word] || seen[word] {
			continue
		}
		seen[word] = true
		terms = append(terms, word)
		if len(terms) >= 12 {
			break
		}
	}
	return terms
}

// foldArabic normalises the character variants people type interchangeably and
// drops the marks that never appear in stored text. Deterministic
// normalisation of this kind is the one place rules belong: it is orthography,
// not meaning.
func foldArabic(r rune) rune {
	switch r {
	case 'أ', 'إ', 'آ', 'ٱ':
		return 'ا'
	case 'ى':
		return 'ي'
	case 'ة':
		return 'ه'
	case 'ؤ':
		return 'و'
	case 'ئ':
		return 'ي'
	case 'ـ': // tatweel
		return ' '
	}
	// Combining marks (fatha, damma, shadda …) carry no search signal.
	if r >= 0x064B && r <= 0x0652 {
		return ' '
	}
	return r
}

// tsQuery builds an OR-ed tsquery from the terms, with each term also matched
// as a prefix so a question about "إجازات" finds a document about "إجازة".
//
// The terms are letters and digits only by construction — searchTerms discards
// everything else — so no tsquery operator can be smuggled in. The query is
// still passed as a bind parameter, never concatenated into SQL.
func tsQuery(terms []string) string {
	parts := make([]string, 0, len(terms))
	for _, t := range terms {
		parts = append(parts, t+":*")
	}
	return strings.Join(parts, " | ")
}

// knowledgeHit is one ranked document.
type knowledgeHit struct {
	Source  string  `json:"source"`
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet,omitempty"`
	Updated string  `json:"updated,omitempty"`
	Score   float64 `json:"-"`
}

// searchHelpDocuments ranks Info Bank documents the caller may read.
//
// The authorization predicates are copied from HelpDocumentRepository.
// SearchDocuments deliberately: this is a different query with the same access
// rules, and the rules are the part that must not drift.
func (d *Deps) searchHelpDocuments(ctx context.Context, actor *Actor, terms []string, limit int) ([]knowledgeHit, error) {
	if actor.DeptID() == nil || len(terms) == 0 {
		return nil, nil
	}
	const vector = `(setweight(to_tsvector('simple', coalesce(doc.title,'')), 'A') ||
	                 setweight(to_tsvector('simple', coalesce(doc.content,'')), 'B'))`

	query := `
		SELECT doc.id, doc.title, LEFT(doc.content, 4000), doc.updated_at,
		       ts_rank_cd(` + vector + `, to_tsquery('simple', $3)) AS score
		FROM help_documents doc
		LEFT JOIN help_document_access acc ON acc.document_id = doc.id AND acc.employee_id = $2
		WHERE doc.department_id = $1
		  AND ` + vector + ` @@ to_tsquery('simple', $3)`
	if actor.Role() != "manager" && actor.Role() != "admin" && !actor.Employee.CanManageHelpDocs {
		query += ` AND COALESCE(acc.access_level, 'read') != 'hide'`
	}
	query += ` ORDER BY score DESC, doc.updated_at DESC LIMIT $4`

	rows, err := d.DB.Query(ctx, query, *actor.DeptID(), actor.ID(), tsQuery(terms), limit)
	if err != nil {
		return nil, fmt.Errorf("search help documents: %w", err)
	}
	defer rows.Close()

	var hits []knowledgeHit
	for rows.Next() {
		var id uuid.UUID
		var title, content string
		var updated time.Time
		var score float64
		if err := rows.Scan(&id, &title, &content, &updated, &score); err != nil {
			return nil, err
		}
		hits = append(hits, knowledgeHit{
			Source:  "help_doc",
			ID:      id.String(),
			Title:   title,
			Snippet: bestSnippet(plainText(content), terms, 320),
			Updated: updated.Format("2006-01-02"),
			Score:   score,
		})
	}
	return hits, rows.Err()
}

// searchFiberxDocuments ranks FiberX Data documents the caller may read,
// including ones shared into their department.
func (d *Deps) searchFiberxDocuments(ctx context.Context, actor *Actor, terms []string, limit int) ([]knowledgeHit, error) {
	if actor.DeptID() == nil || len(terms) == 0 {
		return nil, nil
	}
	const vector = `(setweight(to_tsvector('simple', coalesce(doc.title,'')), 'A') ||
	                 setweight(to_tsvector('simple', coalesce(doc.content,'')), 'B'))`

	query := `
		SELECT doc.id, doc.title, LEFT(doc.content, 4000), doc.updated_at,
		       ts_rank_cd(` + vector + `, to_tsquery('simple', $3)) AS score
		FROM fiberx_data doc
		LEFT JOIN fiberx_data_department_shares shr ON shr.data_id = doc.id AND shr.department_id = $1
		LEFT JOIN fiberx_data_employee_access acc ON acc.data_id = doc.id AND acc.employee_id = $2
		WHERE (doc.department_id = $1 OR shr.id IS NOT NULL)
		  AND ` + vector + ` @@ to_tsquery('simple', $3)`
	if actor.Role() != "manager" && actor.Role() != "admin" && actor.Role() != "team_leader" && !actor.Employee.CanManageFiberxData {
		query += ` AND COALESCE(acc.access_level, 'read') != 'hide'`
	}
	query += ` ORDER BY score DESC, doc.updated_at DESC LIMIT $4`

	rows, err := d.DB.Query(ctx, query, *actor.DeptID(), actor.ID(), tsQuery(terms), limit)
	if err != nil {
		return nil, fmt.Errorf("search fiberx documents: %w", err)
	}
	defer rows.Close()

	var hits []knowledgeHit
	for rows.Next() {
		var id uuid.UUID
		var title, content string
		var updated time.Time
		var score float64
		if err := rows.Scan(&id, &title, &content, &updated, &score); err != nil {
			return nil, err
		}
		hits = append(hits, knowledgeHit{
			Source:  "fiberx_data",
			ID:      id.String(),
			Title:   title,
			Snippet: bestSnippet(plainText(content), terms, 320),
			Updated: updated.Format("2006-01-02"),
			Score:   score,
		})
	}
	return hits, rows.Err()
}

// bestSnippet returns the window of text around the term that matched, so the
// model sees the sentence that made the document a hit rather than whatever
// happens to be at the top of the page.
func bestSnippet(text string, terms []string, width int) string {
	lower := strings.ToLower(text)
	best := -1
	for _, term := range terms {
		if idx := strings.Index(lower, strings.ToLower(term)); idx >= 0 {
			if best < 0 || idx < best {
				best = idx
			}
		}
	}
	if best < 0 {
		return clipRunes(text, width)
	}
	runes := []rune(text)
	start := len([]rune(text[:best])) - width/4
	if start < 0 {
		start = 0
	}
	end := start + width
	if end > len(runes) {
		end = len(runes)
	}
	out := string(runes[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(runes) {
		out += "…"
	}
	return out
}
