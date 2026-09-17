package docx

// tblXML/trXML/tcXML/tcPrXML capture just enough of a <w:tbl>'s structure
// via struct-based unmarshalling (rather than the flat token walk used
// for everything else) since a table's row/cell nesting is naturally
// tree-shaped. Each cell's own content is captured as raw XML and
// re-parsed through parseBlocks, so a cell gets exactly the same marks,
// links and list handling as any other paragraph (SPEC B4).
type tblXML struct {
	Rows []trXML `xml:"tr"`
}

type trXML struct {
	TrPr  trPrXML `xml:"trPr"`
	Cells []tcXML `xml:"tc"`
}

type trPrXML struct {
	TblHeader *struct{} `xml:"tblHeader"`
}

type tcXML struct {
	TcPr     tcPrXML `xml:"tcPr"`
	InnerXML string  `xml:",innerxml"`
}

type tcPrXML struct {
	GridSpan *struct {
		Val int `xml:"val,attr"`
	} `xml:"gridSpan"`
	VMerge *struct {
		Val string `xml:"val,attr"`
	} `xml:"vMerge"`
}

// tableCell is one already-parsed cell: its paragraphs (any nested table
// already flattened into them, SPEC B4), its column span, and whether it
// continues a vertical merge from the row above — real OOXML gives such a
// cell no content of its own, and it's dropped entirely when rendering,
// with its column's rowspan computed on the restarting cell instead.
type tableCell struct {
	colSpan    int
	vMergeKind string // "", "restart", "continue"
	paragraphs []paragraph
}

type tableRow struct {
	header bool
	cells  []tableCell
}

type tableBlock struct {
	rows []tableRow
}

// xmlNamespaceWrapper re-declares every namespace this converter reads
// attributes or elements from. A fragment captured via xml:",innerxml"
// carries none of its own — the declarations live on ancestor elements
// the capture doesn't include — so wrapping it in a fresh root using
// these lets a new decoder over just the fragment resolve prefixes
// exactly as the original document did.
const xmlNamespaceWrapper = `<root ` +
	`xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
	`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
	`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
	`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture" ` +
	`xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" ` +
	`xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math" ` +
	`xmlns:v="urn:schemas-microsoft-com:vml">`

func wrapInnerXML(innerXML string) []byte {
	return []byte(xmlNamespaceWrapper + innerXML + "</root>")
}

// buildTableBlock resolves a struct-decoded <w:tbl> into a tableBlock,
// recursively re-parsing each cell's captured XML. The overall document
// already passed checkXMLDepth before any of this runs, and a subtree can
// never be deeper than the whole it came from, so re-parsing captured
// fragments needs no depth check of its own.
func buildTableBlock(raw tblXML, styles styleSheet, hyperlinkRels map[string]string) (tableBlock, []string, error) {
	var tbl tableBlock
	var notes []string
	for _, r := range raw.Rows {
		row := tableRow{header: r.TrPr.TblHeader != nil}
		for _, c := range r.Cells {
			colSpan := 1
			if c.TcPr.GridSpan != nil && c.TcPr.GridSpan.Val > 0 {
				colSpan = c.TcPr.GridSpan.Val
			}
			vMergeKind := ""
			if c.TcPr.VMerge != nil {
				vMergeKind = c.TcPr.VMerge.Val
				if vMergeKind == "" {
					vMergeKind = "continue" // real OOXML: a bare <w:vMerge/> continues.
				}
			}

			cellDoc, err := parseBlocks(wrapInnerXML(c.InnerXML), styles, hyperlinkRels)
			if err != nil {
				return tableBlock{}, nil, err
			}
			notes = append(notes, cellDoc.notes...)

			row.cells = append(row.cells, tableCell{
				colSpan:    colSpan,
				vMergeKind: vMergeKind,
				paragraphs: flattenBlocksToParagraphs(cellDoc.blocks),
			})
		}
		tbl.rows = append(tbl.rows, row)
	}
	return tbl, notes, nil
}

// flattenBlocksToParagraphs turns a block sequence into plain paragraphs,
// flattening any nested table (SPEC B4: "a table nested in a cell is
// flattened into paragraphs in that cell") by splicing in each of its
// cells' own paragraphs, in reading order.
func flattenBlocksToParagraphs(blocks []block) []paragraph {
	var out []paragraph
	for _, b := range blocks {
		switch b.kind {
		case blockParagraph:
			out = append(out, b.paragraph)
		case blockTable:
			for _, row := range b.table.rows {
				for _, cell := range row.cells {
					out = append(out, cell.paragraphs...)
				}
			}
		}
	}
	return out
}
