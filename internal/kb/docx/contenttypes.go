package docx

type contentTypes struct {
	Overrides []struct {
		PartName    string `xml:"PartName,attr"`
		ContentType string `xml:"ContentType,attr"`
	} `xml:"Override"`
}

// declaresWordDocument reports whether [Content_Types].xml declares a
// WordprocessingML main document part (SPEC B4: "a zip whose
// [Content_Types].xml declares a WordprocessingML main document"). This
// accepts .docm's macro-enabled content type too — SPEC says a .docm is
// read as plain content, never executed.
func (c contentTypes) declaresWordDocument() bool {
	for _, o := range c.Overrides {
		if hasSuffixFold(o.ContentType, "wordprocessingml.document.main+xml") ||
			hasSuffixFold(o.ContentType, "wordprocessingml.document.macroEnabled.main+xml") {
			return true
		}
	}
	return false
}

func parseContentTypes(data []byte) (contentTypes, error) {
	var ct contentTypes
	if err := decodeXML(data, &ct); err != nil {
		return contentTypes{}, err
	}
	return ct, nil
}

type relationship struct {
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

type relationships struct {
	Relationships []relationship `xml:"Relationship"`
}

func parseRelationships(data []byte) ([]relationship, error) {
	var rels relationships
	if err := decodeXML(data, &rels); err != nil {
		return nil, err
	}
	return rels.Relationships, nil
}
