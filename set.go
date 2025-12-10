package main

type Set[T comparable] map[T]struct{}

func (s Set[T]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

func (s Set[T]) Add(v T) {
	s[v] = struct{}{}
}

func (s Set[T]) Remove(v T) {
	delete(s, v)
}

func (s Set[T]) Clear() {
	clear(s)
}

func (s Set[T]) Count() int {
	return len(s)
}

func CreateSet[T comparable](list ...T) Set[T] {
	set := make(Set[T])
	for _, v := range list {
		set.Add(v)
	}
	return set
}
