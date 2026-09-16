package main

func documentXML(bodyContent string, withHeaderFooter bool) string {
	sectPr := "<w:sectPr/>"
	if withHeaderFooter {
		sectPr = `<w:sectPr>` +
			`<w:headerReference w:type="default" r:id="rIdHeader1"/>` +
			`<w:footerReference w:type="default" r:id="rIdFooter1"/>` +
			`</w:sectPr>`
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document
xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"
xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"
xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"
xmlns:v="urn:schemas-microsoft-com:vml">
<w:body>` + bodyContent + sectPr + `</w:body>
</w:document>`
}

type relEntry struct {
	id       string
	typeURI  string
	target   string
	external bool
}

const relNS = "http://schemas.openxmlformats.org/package/2006/relationships"
const officeRelBase = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

func documentRelsXML(entries []relEntry) string {
	out := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<Relationships xmlns="` + relNS + `">` + "\n"
	for _, e := range entries {
		mode := ""
		if e.external {
			mode = ` TargetMode="External"`
		}
		out += `<Relationship Id="` + e.id + `" Type="` + e.typeURI + `" Target="` + e.target + `"` + mode + `/>` + "\n"
	}
	out += `</Relationships>`
	return out
}

func rel(id, kind, target string) relEntry {
	return relEntry{id: id, typeURI: officeRelBase + "/" + kind, target: target}
}

func externalRel(id, kind, target string) relEntry {
	return relEntry{id: id, typeURI: officeRelBase + "/" + kind, target: target, external: true}
}
