package headroom

type CCRStore interface {
	Store(original, compressed string, kind ContentKind) string
	Retrieve(id string) (string, bool)
}
