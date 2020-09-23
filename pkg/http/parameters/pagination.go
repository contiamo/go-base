package parameters

import "strconv"

const (
	// DefaultPageSize is a default size for the pagination
	DefaultPageSize = uint32(20)
	// MinPageSize is a minimal page size for the pagination
	MinPageSize = uint32(5)
	// MinPageSize is a maximal page size for the pagination
	MaxPageSize = uint32(100)
)

// Page contains all the pagination parameters
type Page struct {
	// Number is the page number that starts with 1
	Number,
	// Size is the page size (how many items fits on one page)
	Size uint32
}

// NormalizePagination returns a struct with normalized pagination parameters
func NormalizePagination(pageNumber, pageSize string) Page {
	return NormalizePaginationWithValues(pageNumber, pageSize, DefaultPageSize, MinPageSize, MaxPageSize)
}

// NormalizePaginationWithValues returns a struct with normalized pagination parameters
// according to the settings
func NormalizePaginationWithValues(pageNumber, pageSize string, defaultPageSize, minPageSize, maxPageSize uint32) Page {
	n, _ := strconv.ParseUint(pageNumber, 10, 32) // no need to handle an error, 0 values are fine
	s, _ := strconv.ParseUint(pageSize, 10, 32)

	number := uint32(n)
	size := uint32(s)

	if number < 1 {
		number = 1
	}

	switch {
	case size == 0:
		size = defaultPageSize
	case size < minPageSize:
		size = minPageSize
	case size > maxPageSize:
		size = maxPageSize
	}
	return Page{number, size}
}
