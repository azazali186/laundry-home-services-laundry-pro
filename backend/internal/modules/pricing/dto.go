package pricing

import (
	"github.com/aeroxe/laundry-pro/backend/internal/modules/combo"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/servicetype"
)

// View is the public pricing catalog served to unauthenticated clients
// (landing page, marketing sites). It only contains sellable items.
type View struct {
	BusinessName string             `json:"business_name"`
	Currency     string             `json:"currency"`
	TaxRate      float64            `json:"tax_rate"`
	Services     []servicetype.View `json:"services"`
	Combos       []combo.View       `json:"combos"`
}
