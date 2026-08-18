package pagination

// ColumnType describes the GORM column type used for typed comparisons.
type ColumnType string

const (
	TypeString ColumnType = "string"
	TypeNumber ColumnType = "number"
	TypeTime   ColumnType = "time"
	TypeUUID   ColumnType = "uuid"
	TypeBool   ColumnType = "bool"
)

// Spec describes what a resource permits for sorting/filtering/searching.
// Columns absent from these whitelists can never be used dynamically,
// which prevents injection through ORDER BY / WHERE clauses.
type Spec struct {
	DefaultSort string
	Sortable    map[string]ColumnType
	Filterable  map[string]ColumnType
	Searchable  []string
}

// NewSpec builds a Spec with sensible defaults (id, created_at, updated_at).
func NewSpec(sortable, filterable map[string]ColumnType, searchable ...string) Spec {
	spec := Spec{
		DefaultSort: "created_at",
		Sortable:    map[string]ColumnType{},
		Filterable:  map[string]ColumnType{},
	}
	for k, v := range sortable {
		spec.Sortable[k] = v
	}
	for k, v := range filterable {
		spec.Filterable[k] = v
	}
	spec.Sortable["id"] = TypeUUID
	spec.Sortable["created_at"] = TypeTime
	spec.Sortable["updated_at"] = TypeTime
	spec.Filterable["id"] = TypeUUID
	spec.Filterable["created_at"] = TypeTime
	spec.Filterable["updated_at"] = TypeTime
	spec.Searchable = searchable
	return spec
}

// IsSortable reports whether the column may be used for ordering.
func (s Spec) IsSortable(col string) bool {
	_, ok := s.Sortable[col]
	return ok
}

// IsFilterable reports whether the column may be filtered.
func (s Spec) IsFilterable(col string) bool {
	_, ok := s.Filterable[col]
	return ok
}

// SortType returns the type of a sortable column.
func (s Spec) SortType(col string) ColumnType { return s.Sortable[col] }
