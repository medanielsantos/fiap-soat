package pagination

import (
	"errors"
	"math"
	"net/url"
	"strconv"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

var (
	ErrInvalidPage    = errors.New("page must be a positive integer")
	ErrInvalidPerPage = errors.New("per_page must be between 1 and 100")
)

type Params struct {
	page    int
	perPage int
}

func NewParams(page, perPage int) (Params, error) {
	if page < 1 {
		return Params{}, ErrInvalidPage
	}
	if perPage < 1 || perPage > MaxPerPage {
		return Params{}, ErrInvalidPerPage
	}
	if page-1 > math.MaxInt/perPage {
		return Params{}, ErrInvalidPage
	}
	return Params{page: page, perPage: perPage}, nil
}

func Parse(page, perPage string) (Params, error) {
	parsedPage, parsedPerPage := DefaultPage, DefaultPerPage
	if page != "" {
		value, err := strconv.Atoi(page)
		if err != nil || value < 1 {
			return Params{}, ErrInvalidPage
		}
		parsedPage = value
	}
	if perPage != "" {
		value, err := strconv.Atoi(perPage)
		if err != nil || value < 1 || value > MaxPerPage {
			return Params{}, ErrInvalidPerPage
		}
		parsedPerPage = value
	}
	return NewParams(parsedPage, parsedPerPage)
}

func (params Params) normalized() Params {
	if params.page == 0 && params.perPage == 0 {
		return Params{page: DefaultPage, perPage: DefaultPerPage}
	}
	return params
}

func (params Params) Page() int    { return params.normalized().page }
func (params Params) PerPage() int { return params.normalized().perPage }
func (params Params) Limit() int   { return params.PerPage() }
func (params Params) Offset() int  { return (params.Page() - 1) * params.PerPage() }

type Page[T any] struct {
	Items      []T
	Page       int
	PerPage    int
	TotalItems int64
	TotalPages int64
}

func NewPage[T any](items []T, params Params, totalItems int64) Page[T] {
	if items == nil {
		items = make([]T, 0)
	}
	totalPages := totalItems / int64(params.PerPage())
	if totalItems%int64(params.PerPage()) != 0 {
		totalPages++
	}
	return Page[T]{
		Items:      items,
		Page:       params.Page(),
		PerPage:    params.PerPage(),
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

type Metadata struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalItems int64 `json:"total_items"`
	TotalPages int64 `json:"total_pages"`
}

type Links struct {
	Self string  `json:"self"`
	Next *string `json:"next"`
	Prev *string `json:"prev"`
}

type Response[T any] struct {
	Data       []T      `json:"data"`
	Pagination Metadata `json:"pagination"`
	Links      Links    `json:"links"`
}

func NewResponse[T any](page Page[T], path string) Response[T] {
	self := buildLink(path, page.Page, page.PerPage)
	links := Links{Self: self}
	if int64(page.Page) < page.TotalPages {
		next := buildLink(path, page.Page+1, page.PerPage)
		links.Next = &next
	}
	if page.Page > 1 {
		prev := buildLink(path, page.Page-1, page.PerPage)
		links.Prev = &prev
	}
	return Response[T]{
		Data: page.Items,
		Pagination: Metadata{
			Page: page.Page, PerPage: page.PerPage,
			TotalItems: page.TotalItems, TotalPages: page.TotalPages,
		},
		Links: links,
	}
}

func buildLink(path string, page, perPage int) string {
	parsed, err := url.Parse(path)
	if err != nil {
		return path
	}
	query := parsed.Query()
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
