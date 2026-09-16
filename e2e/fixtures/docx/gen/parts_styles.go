package main

// wordStyles mimics real Word 365 output: style IDs are localised (German
// UI: "berschrift1" for Heading 1, mangling the "Ü" — a real interop trap),
// while style *names* stay the canonical English ones the converter must
// match against (SPEC B4).
const wordStylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:default="1" w:styleId="Normal">
<w:name w:val="Normal"/>
</w:style>
<w:style w:type="paragraph" w:styleId="Title">
<w:name w:val="Title"/>
<w:basedOn w:val="Normal"/>
</w:style>
<w:style w:type="paragraph" w:styleId="berschrift1">
<w:name w:val="heading 1"/>
<w:basedOn w:val="Normal"/>
<w:pPr><w:outlineLvl w:val="0"/></w:pPr>
</w:style>
<w:style w:type="paragraph" w:styleId="berschrift2">
<w:name w:val="heading 2"/>
<w:basedOn w:val="Normal"/>
<w:pPr><w:outlineLvl w:val="1"/></w:pPr>
</w:style>
<w:style w:type="paragraph" w:styleId="berschrift3">
<w:name w:val="heading 3"/>
<w:basedOn w:val="Normal"/>
<w:pPr><w:outlineLvl w:val="2"/></w:pPr>
</w:style>
<w:style w:type="paragraph" w:styleId="Kundenberschrift">
<w:name w:val="Customer Heading"/>
<w:basedOn w:val="berschrift1"/>
</w:style>
<w:style w:type="paragraph" w:styleId="Listenabsatz">
<w:name w:val="List Paragraph"/>
<w:basedOn w:val="Normal"/>
</w:style>
<w:style w:type="character" w:styleId="Internetlink">
<w:name w:val="Hyperlink"/>
</w:style>
</w:styles>`

// googleDocsStylesXML mimics a Google Docs .docx export: style IDs match
// their names directly (no localisation), and Google Docs doesn't emit
// mso- prefixed metadata.
const googleDocsStylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:default="1" w:styleId="Normal">
<w:name w:val="Normal"/>
</w:style>
<w:style w:type="paragraph" w:styleId="Title">
<w:name w:val="Title"/>
<w:basedOn w:val="Normal"/>
</w:style>
<w:style w:type="paragraph" w:styleId="Heading1">
<w:name w:val="heading 1"/>
<w:basedOn w:val="Normal"/>
<w:pPr><w:outlineLvl w:val="0"/></w:pPr>
</w:style>
<w:style w:type="paragraph" w:styleId="Heading2">
<w:name w:val="heading 2"/>
<w:basedOn w:val="Normal"/>
<w:pPr><w:outlineLvl w:val="1"/></w:pPr>
</w:style>
<w:style w:type="paragraph" w:styleId="ListParagraph">
<w:name w:val="List Paragraph"/>
<w:basedOn w:val="Normal"/>
</w:style>
</w:styles>`

// numberingXML defines one bulleted list (numId 1) and one decimal list
// (numId 2), each with two nesting levels, shared by both variants.
const numberingXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:abstractNum w:abstractNumId="0">
<w:lvl w:ilvl="0"><w:numFmt w:val="bullet"/><w:lvlText w:val=""/></w:lvl>
<w:lvl w:ilvl="1"><w:numFmt w:val="bullet"/><w:lvlText w:val="o"/></w:lvl>
</w:abstractNum>
<w:abstractNum w:abstractNumId="1">
<w:lvl w:ilvl="0"><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>
<w:lvl w:ilvl="1"><w:numFmt w:val="lowerLetter"/><w:lvlText w:val="%2."/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
<w:num w:numId="2"><w:abstractNumId w:val="1"/></w:num>
</w:numbering>`
