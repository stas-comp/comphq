package docx

import "strings"

type styleDefXML struct {
	ID   string `xml:"styleId,attr"`
	Type string `xml:"type,attr"`
	Name struct {
		Val string `xml:"val,attr"`
	} `xml:"name"`
	BasedOn struct {
		Val string `xml:"val,attr"`
	} `xml:"basedOn"`
	ParagraphProps paragraphPropsXML `xml:"pPr"`
	RunProps       runPropsXML       `xml:"rPr"`
}

type stylesXML struct {
	Styles []styleDefXML `xml:"style"`
}

// styleSheet resolves a paragraph's styleId to its name and effective
// outline level, following basedOn chains (SPEC B4: "Resolve paragraph and
// character styles through the styles.xml basedOn chains. Match style
// names ... not style IDs, because Word localises IDs.").
type styleSheet struct {
	byID map[string]styleDefXML
}

func parseStyles(data []byte) (styleSheet, error) {
	sheet := styleSheet{byID: map[string]styleDefXML{}}
	if len(data) == 0 {
		return sheet, nil
	}
	var parsed stylesXML
	if err := decodeXML(data, &parsed); err != nil {
		return styleSheet{}, err
	}
	for _, s := range parsed.Styles {
		sheet.byID[s.ID] = s
	}
	return sheet, nil
}

// name returns the style's own name (SPEC: names don't get localised the
// way IDs do, so this is what "heading 1" / "Title" matching is done
// against).
func (s styleSheet) name(styleID string) string {
	return s.byID[styleID].Name.Val
}

// outlineLevel walks the basedOn chain looking for the first w:outlineLvl,
// returning (level, true), or (0, false) if none of the chain sets one.
func (s styleSheet) outlineLevel(styleID string) (int, bool) {
	seen := map[string]bool{}
	for styleID != "" && !seen[styleID] {
		seen[styleID] = true
		def, ok := s.byID[styleID]
		if !ok {
			return 0, false
		}
		if def.ParagraphProps.OutlineLvl != nil {
			return def.ParagraphProps.OutlineLvl.Val, true
		}
		styleID = def.BasedOn.Val
	}
	return 0, false
}

// headingLevel returns the h2/h3 mapping for a paragraph with the given
// direct styleId and (possibly absent) direct outline level, per SPEC B4:
// "heading 1 → h2; heading 2–9 → h3", and "w:outlineLvl also marks a
// heading". outlineLvl 0 is heading 1, 1..8 is heading 2..9.
func (s styleSheet) headingLevel(styleID string, directOutlineLvl *int) (htmlTag string, isHeading bool) {
	outline, has := 0, false
	if directOutlineLvl != nil {
		outline, has = *directOutlineLvl, true
	} else if styleID != "" {
		outline, has = s.outlineLevel(styleID)
	}

	name := strings.ToLower(strings.TrimSpace(s.name(styleID)))
	switch {
	case has && outline == 0:
		return "h2", true
	case has && outline >= 1 && outline <= 8:
		return "h3", true
	case name == "heading 1":
		return "h2", true
	case strings.HasPrefix(name, "heading ") && len(name) > len("heading "):
		return "h3", true
	default:
		return "", false
	}
}

// isTitleStyle reports whether styleID resolves to Word's "Title" style.
func (s styleSheet) isTitleStyle(styleID string) bool {
	return strings.EqualFold(strings.TrimSpace(s.name(styleID)), "Title")
}

// runProps walks styleID's basedOn chain looking for the nearest explicit
// bold/italic in each style's own top-level rPr (SPEC B4: "style
// inheritance"): the first explicit setting found while walking up wins,
// and a chain further up is only consulted for whichever of bold/italic
// the nearer style left unset. Returns nil for either that no style in
// the chain sets at all.
func (s styleSheet) runProps(styleID string) (bold, italic *bool) {
	seen := map[string]bool{}
	for styleID != "" && !seen[styleID] {
		seen[styleID] = true
		def, ok := s.byID[styleID]
		if !ok {
			return bold, italic
		}
		if bold == nil && def.RunProps.Bold != nil {
			v := def.RunProps.Bold.bool()
			bold = &v
		}
		if italic == nil && def.RunProps.Italic != nil {
			v := def.RunProps.Italic.bool()
			italic = &v
		}
		if bold != nil && italic != nil {
			return bold, italic
		}
		styleID = def.BasedOn.Val
	}
	return bold, italic
}

// numbering walks styleID's basedOn chain for the nearest w:numPr in a
// style's own pPr (SPEC B4: "a list defined via paragraph style"),
// returning ok=false if nothing in the chain sets one.
func (s styleSheet) numbering(styleID string) (numID, ilvl int, ok bool) {
	seen := map[string]bool{}
	for styleID != "" && !seen[styleID] {
		seen[styleID] = true
		def, has := s.byID[styleID]
		if !has {
			return 0, 0, false
		}
		if def.ParagraphProps.NumPr != nil {
			if def.ParagraphProps.NumPr.NumID != nil {
				numID = def.ParagraphProps.NumPr.NumID.Val
			}
			if def.ParagraphProps.NumPr.Ilvl != nil {
				ilvl = def.ParagraphProps.NumPr.Ilvl.Val
			}
			return numID, ilvl, true
		}
		styleID = def.BasedOn.Val
	}
	return 0, 0, false
}

// resolveMarks combines a run's direct bold/italic (nil if the run's own
// rPr doesn't mention it) with its paragraph style's and, if set, its own
// character style's (SPEC B4: "bold/italic toggles ... honouring
// w:val=0/false and style inheritance"). Precedence, weakest first:
// paragraph style, then character style (w:rStyle), then the run's own
// direct formatting — each level only overrides what the levels below it
// left unset, except the run's own direct formatting, which always wins
// when present, including to explicitly turn something off.
func (s styleSheet) resolveMarks(paragraphStyleID, runStyleID string, directBold, directItalic *bool) (bold, italic bool) {
	pBold, pItalic := s.runProps(paragraphStyleID)
	if pBold != nil {
		bold = *pBold
	}
	if pItalic != nil {
		italic = *pItalic
	}
	if runStyleID != "" {
		cBold, cItalic := s.runProps(runStyleID)
		if cBold != nil {
			bold = *cBold
		}
		if cItalic != nil {
			italic = *cItalic
		}
	}
	if directBold != nil {
		bold = *directBold
	}
	if directItalic != nil {
		italic = *directItalic
	}
	return bold, italic
}
