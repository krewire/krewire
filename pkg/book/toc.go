package book

// tocEntry is one row in the sidebar chapter list. HasSubs indicates a
// collapsible group; Open indicates it should be expanded initially (active
// path). Subs holds all subchapters for that group.
type tocEntry struct {
	Label   string
	Title   string
	Link    string
	Active  bool
	HasSubs bool
	Open    bool
	Subs    []tocEntry
}

// buildTOCEntry builds a tocEntry and all its nested subchapters recursively.
func (b *Book) buildTOCEntry(ch *Chapter, active *Chapter, isTop bool) (tocEntry, bool) {
	hasSubs := len(ch.Subs) > 0
	activeSelf := ch == active
	activeInSubs := false
	subs := make([]tocEntry, 0, len(ch.Subs))
	for _, s := range ch.Subs {
		subEntry, subActive := b.buildTOCEntry(s, active, false)
		if subActive {
			activeInSubs = true
		}
		subs = append(subs, subEntry)
	}
	open := activeSelf || activeInSubs
	label := ch.Label()
	if isTop {
		label += "."
	}
	return tocEntry{
		Label:   label,
		Title:   ch.Title,
		Link:    b.link(ch.Path()),
		Active:  activeSelf,
		HasSubs: hasSubs,
		Open:    open,
		Subs:    subs,
	}, open
}

// chapterList builds the sidebar chapter list for responsive collapsible nav.
// Every chapter with subs gets a collapsible group; Open is true when the
// group contains the active page (itself or any sub). All groups are rendered
// so the client can toggle them without a round-trip — CSS handles the
// collapsed state, JS persists it.
func (b *Book) chapterList(active *Chapter) []tocEntry {
	toc := make([]tocEntry, 0, len(b.Chapters))
	for i := range b.Chapters {
		entry, _ := b.buildTOCEntry(&b.Chapters[i], active, true)
		toc = append(toc, entry)
	}
	return toc
}
