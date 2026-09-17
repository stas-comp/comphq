package docx

import "strings"

type numLvlXML struct {
	Ilvl   int `xml:"ilvl,attr"`
	NumFmt struct {
		Val string `xml:"val,attr"`
	} `xml:"numFmt"`
}

type abstractNumXML struct {
	ID     int         `xml:"abstractNumId,attr"`
	Levels []numLvlXML `xml:"lvl"`
}

type numDefXML struct {
	NumID         int `xml:"numId,attr"`
	AbstractNumID struct {
		Val int `xml:"val,attr"`
	} `xml:"abstractNumId"`
}

type numberingFileXML struct {
	AbstractNums []abstractNumXML `xml:"abstractNum"`
	Nums         []numDefXML      `xml:"num"`
}

// numberingSheet resolves a (numId, ilvl) pair to whether that level is a
// bulleted list (SPEC B4: "a numbering.xml level with numFmt bullet ->
// ul; any other format -> ol").
type numberingSheet struct {
	abstractByID  map[int]abstractNumXML
	numToAbstract map[int]int
}

func parseNumbering(data []byte) (numberingSheet, error) {
	sheet := numberingSheet{abstractByID: map[int]abstractNumXML{}, numToAbstract: map[int]int{}}
	if len(data) == 0 {
		return sheet, nil
	}
	var parsed numberingFileXML
	if err := decodeXML(data, &parsed); err != nil {
		return numberingSheet{}, err
	}
	for _, a := range parsed.AbstractNums {
		sheet.abstractByID[a.ID] = a
	}
	for _, n := range parsed.Nums {
		sheet.numToAbstract[n.NumID] = n.AbstractNumID.Val
	}
	return sheet, nil
}

// isBullet reports whether numID's level ilvl uses a bullet format. A
// numId or level this sheet doesn't recognise defaults to false (ol),
// the more common and less visually surprising fallback.
func (n numberingSheet) isBullet(numID, ilvl int) bool {
	abstractID, ok := n.numToAbstract[numID]
	if !ok {
		return false
	}
	abstract, ok := n.abstractByID[abstractID]
	if !ok {
		return false
	}
	for _, lvl := range abstract.Levels {
		if lvl.Ilvl == ilvl {
			return strings.EqualFold(lvl.NumFmt.Val, "bullet")
		}
	}
	return false
}
