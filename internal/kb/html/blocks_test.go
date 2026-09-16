package html

import (
	"reflect"
	"testing"
)

func TestAssignBlocksNumbersInDocumentOrder(t *testing.T) {
	in := `<h2>Title</h2><p>First</p><ul><li>one</li><li>two</li></ul><blockquote>Q</blockquote>` +
		`<table><tr><td>a</td></tr></table><figure><figcaption>Cap</figcaption></figure>`
	want := `<h2 data-b="1">Title</h2><p data-b="2">First</p><ul><li data-b="3">one</li><li data-b="4">two</li></ul>` +
		`<blockquote data-b="5">Q</blockquote><table><tbody><tr data-b="6"><td>a</td></tr></tbody></table>` +
		`<figure><figcaption data-b="7">Cap</figcaption></figure>`

	got := AssignBlocks(in)
	if got != want {
		t.Errorf("AssignBlocks(%q) =\n%q\nwant\n%q", in, got, want)
	}
}

func TestAssignBlocksGivesNestedListItemsTheirOwnID(t *testing.T) {
	in := `<ul><li>outer<ul><li>inner</li></ul></li></ul>`
	want := `<ul><li data-b="1">outer<ul><li data-b="2">inner</li></ul></li></ul>`

	got := AssignBlocks(in)
	if got != want {
		t.Errorf("AssignBlocks(%q) =\n%q\nwant\n%q", in, got, want)
	}
}

func TestBlockTexts(t *testing.T) {
	withIDs := AssignBlocks(`<h2>Title</h2><p>Hello <strong>world</strong></p><ul><li>one</li><li>two</li></ul>`)

	got := BlockTexts(withIDs)
	want := map[int]string{
		1: "Title",
		2: "Hello world",
		3: "one",
		4: "two",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BlockTexts = %v, want %v", got, want)
	}
}

func TestBlockTextsKeepsNestedListItemTextSeparate(t *testing.T) {
	withIDs := AssignBlocks(`<ul><li>outer<ul><li>inner</li></ul></li></ul>`)

	got := BlockTexts(withIDs)
	want := map[int]string{
		1: "outer",
		2: "inner",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BlockTexts = %v, want %v", got, want)
	}
}
