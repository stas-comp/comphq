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
