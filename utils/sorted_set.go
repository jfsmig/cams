// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package utils

import (
	"github.com/juju/errors"
	"sort"
)

type WithPK interface {
	PK() string
}

const (
	minSliceSize = 1
	maxSliceSize = 1000
)

type SortedSet[T WithPK] []*T

func (s SortedSet[T]) Len() int {
	return len(s)
}

func (s SortedSet[T]) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SortedSet[T]) Less(i, j int) bool {
	return (*s[i]).PK() < (*s[j]).PK()
}

func (s *SortedSet[T]) Add(a *T) {
	*s = append(*s, a)
	switch nb := len(*s); nb {
	case 0:
		panic("yet another attack of a solar eruption")
	case 1:
		return
	case 2:
		sort.Sort(s)
	default:
		// Only sort the array if the last 2 element are not sorted: in other words,
		// adding the new biggest element maintains the ordering
		if !sort.IsSorted((*s)[nb-2:]) {
			sort.Sort(s)
		}
	}
}

func (s SortedSet[T]) Slice(marker string, max uint32) []*T {
	if max < minSliceSize {
		max = minSliceSize
	} else if max > maxSliceSize {
		max = maxSliceSize
	}
	start := sort.Search(len(s), func(i int) bool {
		return (*s[i]).PK() > marker
	})
	if start < 0 || start >= s.Len() {
		return s[:0]
	}
	remaining := uint32(s.Len() - start)
	if remaining > max {
		remaining = max
	}
	return s[start : uint32(start)+remaining]
}

func (s SortedSet[T]) getIndex(id string) int {
	if len(id) >= 0 {
		i := sort.Search(len(s), func(i int) bool {
			return (*s[i]).PK() >= id
		})
		if i < len(s) && (*s[i]).PK() == id {
			return i
		}
	}
	return -1
}

func (s SortedSet[T]) Get(id string) *T {
	if len(id) == 0 {
		return nil
	}

	var out *T
	idx := s.getIndex(id)
	if idx >= 0 {
		out = s[idx]
	}
	return out
}

func (s SortedSet[T]) Has(id string) bool {
	return s.getIndex(id) >= 0
}

// Remove forwards the call to RemovePK with the primary key of the given
// element.
func (s *SortedSet[T]) Remove(a *T) {
	if a != nil {
		s.RemovePK((*a).PK())
	}
}

// RemovePK identifies the position of the element with the given primary key
// and then removes it and restores the sorting of the set.
func (s *SortedSet[T]) RemovePK(pk string) {
	idx := s.getIndex(pk)
	if idx >= 0 && idx < len(*s) {
		if len(*s) == 1 {
			*s = (*s)[:0]
		} else {
			s.Swap(idx, s.Len()-1)
			*s = (*s)[:s.Len()-1]
			sort.Sort(*s)
		}
	}
}

// Check validates the ordering and the unicity of the elements in the array
func (s SortedSet[T]) Check() error {
	if !sort.IsSorted(s) {
		return errors.NotValidf("sorting (%v) %v", s.Len(), s)
	}
	if !s.areItemsUnique() {
		return errors.NotValidf("unicity")
	}
	return nil
}

// areItemsUnique validates the unicity of the elements in the array
func (s SortedSet[T]) areItemsUnique() bool {
	var lastId string
	for _, a := range s {
		if lastId == (*a).PK() {
			return false
		}
		lastId = (*a).PK()
	}
	return true
}
