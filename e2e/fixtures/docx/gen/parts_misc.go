package main

const headerXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:p><w:r><w:t xml:space="preserve">Comp HQ — Printer Supplies Handbook</w:t></w:r></w:p>
</w:hdr>`

const footerXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:p><w:r><w:t xml:space="preserve">Internal use only</w:t></w:r></w:p>
</w:ftr>`

const commentsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:comment w:id="0" w:author="Sam" w:date="2026-01-01T00:00:00Z">
<w:p><w:r><w:t xml:space="preserve">Double check this figure before publishing.</w:t></w:r></w:p>
</w:comment>
</w:comments>`

const footnotesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:footnotes xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:footnote w:type="separator" w:id="-1"><w:p><w:r><w:separator/></w:r></w:p></w:footnote>
<w:footnote w:type="continuationSeparator" w:id="0"><w:p><w:r><w:continuationSeparator/></w:r></w:p></w:footnote>
<w:footnote w:id="1">
<w:p><w:r><w:t xml:space="preserve">Prices checked January 2026.</w:t></w:r></w:p>
</w:footnote>
</w:footnotes>`

// A minimal but valid inline-data bar chart. Real Word charts also embed a
// backing worksheet; that's optional and skipped here since Convert()
// treats any chart as an unsupported-item placeholder either way.
const chart1XML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<c:chartSpace xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
<c:chart>
<c:plotArea>
<c:barChart>
<c:barDir val="col"/>
<c:ser>
<c:idx val="0"/>
<c:order val="0"/>
<c:cat><c:strRef><c:f>Sheet1!$A$1:$A$2</c:f><c:strCache>
<c:pt idx="0"><c:v>Jan</c:v></c:pt><c:pt idx="1"><c:v>Feb</c:v></c:pt>
</c:strCache></c:strRef></c:cat>
<c:val><c:numRef><c:f>Sheet1!$B$1:$B$2</c:f><c:numCache>
<c:pt idx="0"><c:v>4</c:v></c:pt><c:pt idx="1"><c:v>7</c:v></c:pt>
</c:numCache></c:numRef></c:val>
</c:ser>
</c:barChart>
</c:plotArea>
</c:chart>
</c:chartSpace>`

// A minimal, real ENHMETAHEADER so the fixture is a recognisable EMF file
// by signature, standing in for a picture format the converter can't
// render (SPEC B4 lists EMF among "pictures in other formats").
var emfBytes = buildMinimalEMF()

func buildMinimalEMF() []byte {
	b := make([]byte, 88)
	putU32 := func(off int, v uint32) {
		b[off] = byte(v)
		b[off+1] = byte(v >> 8)
		b[off+2] = byte(v >> 16)
		b[off+3] = byte(v >> 24)
	}
	putU32(0, 1)           // iType: EMR_HEADER
	putU32(4, 88)          // nSize
	putU32(8, 0)           // rclBounds.left
	putU32(12, 0)          // rclBounds.top
	putU32(16, 100)        // rclBounds.right
	putU32(20, 100)        // rclBounds.bottom
	putU32(24, 0)          // rclFrame.left
	putU32(28, 0)          // rclFrame.top
	putU32(32, 1000)       // rclFrame.right
	putU32(36, 1000)       // rclFrame.bottom
	putU32(40, 0x464D4520) // dSignature "EMF "
	putU32(44, 0x00010000) // nVersion
	putU32(48, 88)         // nBytes
	putU32(52, 1)          // nRecords
	return b
}
