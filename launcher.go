package bannerlord

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Mode selects the singleplayer or multiplayer mod list of LauncherData.xml.
type Mode string

// The two lists the launcher keeps.
const (
	Singleplayer Mode = "SingleplayerData"
	Multiplayer  Mode = "MultiplayerData"
)

// LauncherMod is one ModDatas entry: a module id and whether the launcher enables it.
type LauncherMod struct {
	ID       string
	Selected bool
}

// Launcher is a parsed LauncherData.xml. Elements this package does not model survive a
// round trip; comments and namespace prefixes do not.
type Launcher struct {
	root *node
}

// ParseLauncher parses LauncherData.xml. Every module id in it must be valid.
func ParseLauncher(r io.Reader) (*Launcher, error) {
	root, err := parseTree(r)
	if err != nil {
		return nil, err
	}
	if root.name != "UserData" {
		return nil, fmt.Errorf("%w: root element %q, want UserData", ErrInvalid, root.name)
	}
	l := &Launcher{root: root}
	for _, mode := range []Mode{Singleplayer, Multiplayer} {
		for _, e := range l.entries(mode) {
			if id := e.textOf("Id"); !validID(id) {
				return nil, fmt.Errorf("%w: launcher module id %q", ErrInvalid, id)
			}
		}
	}
	return l, nil
}

func (l *Launcher) entries(mode Mode) []*node {
	data := l.root.child(string(mode))
	if data == nil {
		return nil
	}
	mods := data.child("ModDatas")
	if mods == nil {
		return nil
	}
	return mods.childrenNamed("UserModData")
}

// Mods returns the mode's mod list in launcher order.
func (l *Launcher) Mods(mode Mode) []LauncherMod {
	var out []LauncherMod
	for _, e := range l.entries(mode) {
		id := e.textOf("Id")
		out = append(out, LauncherMod{ID: id, Selected: strings.EqualFold(e.textOf("IsSelected"), "true")})
	}
	return out
}

// SetMods makes enabled the mode's selected set, in that order, at the front of the list. Entries
// already present keep their other fields; entries not named stay, unselected, after the enabled
// ones. Ids must be valid and unique.
func (l *Launcher) SetMods(mode Mode, enabled []string) error {
	want := make(map[string]bool, len(enabled))
	for _, id := range enabled {
		if !validID(id) {
			return fmt.Errorf("%w: module id %q", ErrInvalid, id)
		}
		if want[id] {
			return fmt.Errorf("%w: module id %q listed twice", ErrInvalid, id)
		}
		want[id] = true
	}
	existing := map[string]*node{}
	var rest []*node
	for _, e := range l.entries(mode) {
		id := e.textOf("Id")
		if want[id] {
			existing[id] = e
		} else {
			setText(e, "IsSelected", "false")
			rest = append(rest, e)
		}
	}
	list := make([]*node, 0, len(enabled)+len(rest))
	for _, id := range enabled {
		e := existing[id]
		if e == nil {
			e = &node{name: "UserModData"}
			setText(e, "Id", id)
		}
		setText(e, "IsSelected", "true")
		list = append(list, e)
	}
	list = append(list, rest...)

	data := l.root.child(string(mode))
	if data == nil {
		data = &node{name: string(mode)}
		l.root.kids = append(l.root.kids, data)
	}
	mods := data.child("ModDatas")
	if mods == nil {
		mods = &node{name: "ModDatas"}
		data.kids = append(data.kids, mods)
	}
	kept := mods.kids[:0:0]
	for _, k := range mods.kids {
		if k.name != "UserModData" {
			kept = append(kept, k)
		}
	}
	kept = append(kept, list...)
	mods.kids = kept
	return nil
}

func setText(n *node, name, value string) {
	c := n.child(name)
	if c == nil {
		c = &node{name: name}
		n.kids = append(n.kids, c)
	}
	c.text = value
}

// Marshal renders the launcher data as indented XML with a declaration.
func (l *Launcher) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := l.root.write(enc); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("bannerlord: write xml: %w", err)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
