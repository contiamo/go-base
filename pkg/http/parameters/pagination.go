package parameters

import "strconv"

const (
	defaultPageSize = 20
	minPageSize     = 5
	maxPageSize     = 100
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
	number, _ := strconv.Atoi(pageNumber) // no need to handle an error, 0 values are fine
	size, _ := strconv.Atoi(pageSize)

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
	return Page{uint32(number), uint32(size)}
}
