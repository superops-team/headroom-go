package headroom

type ContentKind int

const (
	KindText        ContentKind = 0
	KindJSON        ContentKind = 1
	KindCode        ContentKind = 2
	KindDiff        ContentKind = 3
	KindLog         ContentKind = 4
	KindSearch      ContentKind = 5
	KindTabular     ContentKind = 6
	KindSpreadsheet ContentKind = 7
	KindHTML        ContentKind = 8
	KindUnknown     ContentKind = 9
)

func (k ContentKind) String() string {
	switch k {
	case KindText:
		return "Text"
	case KindJSON:
		return "JSON"
	case KindCode:
		return "Code"
	case KindDiff:
		return "Diff"
	case KindLog:
		return "Log"
	case KindSearch:
		return "Search"
	case KindTabular:
		return "Tabular"
	case KindSpreadsheet:
		return "Spreadsheet"
	case KindHTML:
		return "HTML"
	case KindUnknown:
		return "Unknown"
	default:
		return "Text"
	}
}
