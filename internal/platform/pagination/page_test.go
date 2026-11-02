package pagination

import "testing"

func TestNewPageComputesTotalsAndNeverReturnsNilItems(t *testing.T) {
	params, _ := NewParams(1, 10)
	cases := []struct {
		total, wantPages int64
	}{{0, 0}, {10, 1}, {11, 2}}
	for _, tc := range cases {
		page := NewPage[int](nil, params, tc.total)
		if page.Items == nil || len(page.Items) != 0 {
			t.Fatalf("items = %#v, want empty slice", page.Items)
		}
		if page.TotalPages != tc.wantPages || page.TotalItems != tc.total || page.Page != 1 || page.PerPage != 10 {
			t.Fatalf("NewPage(total=%d) = %+v, want %d pages", tc.total, page, tc.wantPages)
		}
	}
}

func TestNewResponseBuildsNavigationLinks(t *testing.T) {
	params, _ := NewParams(2, 10)
	response := NewResponse(NewPage([]string{"a"}, params, 30), "/api/v1/items?status=active")

	if response.Links.Self != "/api/v1/items?page=2&per_page=10&status=active" {
		t.Errorf("self = %q", response.Links.Self)
	}
	if response.Links.Next == nil || *response.Links.Next != "/api/v1/items?page=3&per_page=10&status=active" {
		t.Errorf("next = %v", response.Links.Next)
	}
	if response.Links.Prev == nil || *response.Links.Prev != "/api/v1/items?page=1&per_page=10&status=active" {
		t.Errorf("prev = %v", response.Links.Prev)
	}
	want := Metadata{Page: 2, PerPage: 10, TotalItems: 30, TotalPages: 3}
	if response.Pagination != want || len(response.Data) != 1 {
		t.Errorf("response = %+v", response)
	}
}

func TestNewResponseOmitsLinksAtEdges(t *testing.T) {
	params, _ := NewParams(1, 10)
	response := NewResponse(NewPage([]string{"a"}, params, 5), "/items")
	if response.Links.Next != nil || response.Links.Prev != nil {
		t.Fatalf("single page must not link next/prev: %+v", response.Links)
	}
}

func TestBuildLinkReturnsPathWhenUnparseable(t *testing.T) {
	if got := buildLink("://bad\x7f", 1, 10); got != "://bad\x7f" {
		t.Fatalf("buildLink(invalid) = %q", got)
	}
}
