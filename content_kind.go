package headroom

type ContentKind int

const (
	KindText ContentKind = 0
	KindJSON ContentKind = 1
	KindCode ContentKind = 2
)

func (k ContentKind) String() string {
	switch k {
	case KindJSON:
		return "JSON"
	case KindCode:
		return "Code"
	default:
		return "Text"
	}
}
