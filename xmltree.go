package bannerlord

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// MaxXMLSize is the largest XML document the package reads, in bytes.
const MaxXMLSize = 1 << 20

const maxXMLDepth = 32

var (
	// ErrTooLarge reports a document over MaxXMLSize.
	ErrTooLarge = errors.New("bannerlord: xml document too large")
	// ErrDoctype reports a DOCTYPE or other directive, which is never accepted.
	ErrDoctype = errors.New("bannerlord: xml directives are not accepted")
	// ErrInvalid reports a document that is well-formed XML but not the expected shape.
	ErrInvalid = errors.New("bannerlord: invalid document")
)

// node is a minimal ordered XML tree. Namespaces are not kept: Bannerlord files use none.
type node struct {
	name  string
	attrs []xml.Attr
	kids  []*node
	text  string
	// pending collects an open element's text; comments split it into many fragments, and appending to a string
	// each time would copy the whole of it again.
	pending []byte
}

func parseTree(r io.Reader) (*node, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxXMLSize+1))
	if err != nil {
		return nil, fmt.Errorf("bannerlord: read xml: %w", err)
	}
	if len(data) > MaxXMLSize {
		return nil, ErrTooLarge
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	dec := xml.NewDecoder(bytes.NewReader(data))
	var (
		root  *node
		stack []*node
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bannerlord: parse xml: %w", err)
		}
		switch t := tok.(type) {
		case xml.Directive:
			return nil, ErrDoctype
		case xml.StartElement:
			if len(stack) >= maxXMLDepth {
				return nil, fmt.Errorf("%w: nesting deeper than %d", ErrInvalid, maxXMLDepth)
			}
			n := &node{name: t.Name.Local}
			for _, a := range t.Attr {
				n.attrs = append(n.attrs, xml.Attr{Name: xml.Name{Local: a.Name.Local}, Value: a.Value})
			}
			if len(stack) == 0 {
				if root != nil {
					return nil, fmt.Errorf("%w: multiple root elements", ErrInvalid)
				}
				root = n
			} else {
				p := stack[len(stack)-1]
				p.kids = append(p.kids, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			top := stack[len(stack)-1]
			top.text, top.pending = string(top.pending), nil
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				top.pending = append(top.pending, t...)
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("%w: empty document", ErrInvalid)
	}
	return root, nil
}

func (n *node) attr(name string) (string, bool) {
	for _, a := range n.attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

func (n *node) child(name string) *node {
	for _, k := range n.kids {
		if k.name == name {
			return k
		}
	}
	return nil
}

func (n *node) childrenNamed(name string) []*node {
	var out []*node
	for _, k := range n.kids {
		if k.name == name {
			out = append(out, k)
		}
	}
	return out
}

func (n *node) textOf(name string) string {
	if c := n.child(name); c != nil {
		return strings.TrimSpace(c.text)
	}
	return ""
}

// valueOf returns the `value` attribute of the named child, the shape SubModule.xml uses for scalars.
func (n *node) valueOf(name string) (string, bool) {
	c := n.child(name)
	if c == nil {
		return "", false
	}
	return c.attr("value")
}

func (n *node) write(enc *xml.Encoder) error {
	start := xml.StartElement{Name: xml.Name{Local: n.name}, Attr: n.attrs}
	if err := enc.EncodeToken(start); err != nil {
		return fmt.Errorf("bannerlord: write xml: %w", err)
	}
	if len(n.kids) == 0 {
		if t := strings.TrimSpace(n.text); t != "" {
			if err := enc.EncodeToken(xml.CharData(t)); err != nil {
				return fmt.Errorf("bannerlord: write xml: %w", err)
			}
		}
	}
	for _, k := range n.kids {
		if err := k.write(enc); err != nil {
			return err
		}
	}
	if err := enc.EncodeToken(start.End()); err != nil {
		return fmt.Errorf("bannerlord: write xml: %w", err)
	}
	return nil
}

func validID(id string) bool {
	if id == "" || id == "." || id == ".." || len(id) > maxIDLen {
		return false
	}
	for _, c := range id {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-'
		if !ok {
			return false
		}
	}
	return true
}

const maxIDLen = 128
