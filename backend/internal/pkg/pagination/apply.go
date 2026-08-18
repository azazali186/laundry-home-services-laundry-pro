package pagination

import (
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PageResult summarizes a paged query result.
type PageResult struct {
	Total   int64
	Limit   int
	Offset  int
	HasMore bool
}

// Page applies whitelisted filters, search, dynamic sorting and offset/limit
// to base and fills dest (a *[]T) with the current page.
//
// Usage:
//
//	var items []models.Order
//	res, err := pagination.Page(tx, req, orderSpec, &items)
func Page[T any](base *gorm.DB, req PageRequest, spec Spec, dest *[]T) (*PageResult, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	if !spec.IsSortable(req.SortBy) {
		return nil, errors.New(400, "sort column not allowed: "+req.SortBy)
	}

	filtered, err := FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := filtered.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	query := filtered.Session(&gorm.Session{})
	query = query.Offset(req.Offset).Limit(req.Limit)

	dir := strings.ToUpper(req.SortOrder)
	query = query.Order(quoteCol(req.SortBy) + " " + dir)
	if req.SortBy != "id" {
		query = query.Order(`"id" ` + dir)
	}

	if err := query.Find(dest).Error; err != nil {
		return nil, err
	}

	items := *dest
	return &PageResult{
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
		HasMore: int64(req.Offset+len(items)) < total,
	}, nil
}

// FilteredQuery returns a session with whitelisted filters + search applied
// but no ordering or paging. Useful for summary queries.
func FilteredQuery(base *gorm.DB, req PageRequest, spec Spec) (*gorm.DB, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	tx := base.Session(&gorm.Session{})

	for col, raw := range req.Filters {
		baseCol := col
		op := OpEq
		value := raw

		lower := strings.ToLower(col)
		if strings.HasSuffix(lower, "_from") || strings.HasSuffix(lower, "_to") {
			baseCol = strings.TrimSuffix(strings.TrimSuffix(lower, "_from"), "_to")
			if !spec.IsFilterable(baseCol) {
				return nil, errors.New(400, "filter column not allowed: "+baseCol)
			}
			if strings.HasSuffix(lower, "_from") {
				op = OpGte
			} else {
				op = OpLte
			}
			var err error
			tx, err = applyFilter(tx, baseCol, spec, op, value)
			if err != nil {
				return nil, err
			}
			continue
		}

		if !spec.IsFilterable(col) {
			return nil, errors.New(400, "filter column not allowed: "+col)
		}
		for _, prefix := range filterPrefixes {
			if strings.HasPrefix(raw, prefix) {
				op = prefix
				value = strings.TrimPrefix(raw, prefix)
				break
			}
		}
		var err error
		tx, err = applyFilter(tx, baseCol, spec, op, value)
		if err != nil {
			return nil, err
		}
	}

	if req.Search != "" && len(spec.Searchable) > 0 {
		like := "%" + req.Search + "%"
		var conds []string
		var args []any
		for _, col := range spec.Searchable {
			conds = append(conds, quoteCol(col)+" LIKE ?")
			args = append(args, like)
		}
		tx = tx.Where("("+strings.Join(conds, " OR ")+")", args...)
	}
	return tx, nil
}

func applyFilter(tx *gorm.DB, col string, spec Spec, op, value string) (*gorm.DB, error) {
	colType := spec.Filterable[col]
	if op == OpLike {
		return tx.Where(quoteCol(col)+" LIKE ?", "%"+value+"%"), nil
	}
	typed, err := castValue(colType, value)
	if err != nil {
		return nil, errors.New(400, "invalid filter value for "+col)
	}
	if typed == nil {
		return tx.Where(quoteCol(col) + " IS NULL"), nil
	}
	return tx.Where(quoteCol(col)+" "+comparatorFor(op)+" ?", typed), nil
}

func castValue(colType ColumnType, value string) (any, error) {
	switch colType {
	case TypeNumber:
		return parseNumber(value)
	case TypeTime:
		return parseTime(value)
	case TypeUUID:
		return uuid.Parse(value)
	case TypeBool:
		return parseBool(value)
	default:
		if value == "null" {
			return nil, nil
		}
		return value, nil
	}
}

func comparatorFor(op string) string {
	switch op {
	case OpGt:
		return ">"
	case OpGte:
		return ">="
	case OpLt:
		return "<"
	case OpLte:
		return "<="
	default:
		return "="
	}
}

func quoteCol(col string) string { return `"` + col + `"` }
