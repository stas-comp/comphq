package docx

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

var errXMLTooDeep = errors.New("xml element depth exceeds the limit")

// decodeXML unmarshals data into v, first rejecting XML nested deeper than
// MaxXMLDepth (SPEC B4's "deep.docx" limit) so a hostile file can't blow
// the stack or exhaust memory via unmarshal's own recursion.
func decodeXML(data []byte, v any) error {
	if err := checkXMLDepth(data); err != nil {
		if errors.Is(err, errXMLTooDeep) {
			return ErrTooLarge
		}
		return ErrUnreadable
	}
	if err := xml.Unmarshal(data, v); err != nil {
		return ErrUnreadable
	}
	return nil
}

func checkXMLDepth(data []byte) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
			if depth > MaxXMLDepth {
				return errXMLTooDeep
			}
		case xml.EndElement:
			depth--
		}
	}
}
