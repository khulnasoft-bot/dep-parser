package set

type Set[T comparable] map[T]struct{}

func New[T comparable](elems ...T) Set[T] {
	s := make(Set[T], len(elems))
	for _, e := range elems {
		s[e] = struct{}{}
	}
	return s
}

func (s Set[T]) Clone() Set[T] {
	clone := make(Set[T], len(s))
	for e := range s {
		clone[e] = struct{}{}
	}
	return clone
}

func (s Set[T]) Append(elems ...T) {
	for _, e := range elems {
		s[e] = struct{}{}
	}
}

func (s Set[T]) Items() []T {
	items := make([]T, 0, len(s))
	for e := range s {
		items = append(items, e)
	}
	return items
}

func (s Set[T]) Contains(elem T) bool {
	_, ok := s[elem]
	return ok
}

func (s Set[T]) Size() int {
	return len(s)
}

func (s Set[T]) Iter() []T {
	return s.Items()
}
