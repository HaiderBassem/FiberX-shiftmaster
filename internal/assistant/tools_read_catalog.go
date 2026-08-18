package assistant

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
)

// Service catalog tools. Visibility follows the catalog's own scoping rule:
// provinces owned by or shared with the caller's department (admins see all).
// Every price, speed and availability answer comes from these rows — the
// model is instructed to never state a price it did not just read here.

// visibleProvinces resolves the actor's province scope.
func visibleProvinces(ctx context.Context, d *Deps, actor *Actor) ([]models.Province, error) {
	if actor.Role() == "admin" {
		// Admins: union across all departments. The province list per
		// department is small (Iraq has 18 governorates), departments are
		// bounded, and results are deduplicated.
		depts, err := d.DepartmentRepo.GetAll(ctx)
		if err != nil {
			return nil, err
		}
		seen := map[uuid.UUID]bool{}
		var out []models.Province
		for _, dept := range depts {
			provs, err := d.ProvinceService.GetAll(ctx, dept.ID)
			if err != nil {
				continue
			}
			for _, p := range provs {
				if !seen[p.ID] {
					seen[p.ID] = true
					out = append(out, p)
				}
			}
		}
		return out, nil
	}
	if actor.DeptID() == nil {
		return nil, Errf("you are not assigned to a department")
	}
	return d.ProvinceService.GetAll(ctx, *actor.DeptID())
}

// speedMbps extracts a leading number from a free-text speed like "50 Mbps".
func speedMbps(speed *string) float64 {
	if speed == nil {
		return 0
	}
	fields := strings.FieldsFunc(*speed, func(r rune) bool {
		return (r < '0' || r > '9') && r != '.'
	})
	for _, f := range fields {
		if v, err := strconv.ParseFloat(f, 64); err == nil && v > 0 {
			return v
		}
	}
	return 0
}

type planView struct {
	PlanID       string  `json:"plan_id"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	Province     string  `json:"province"`
	Price        float64 `json:"price_iqd"`
	DurationDays int     `json:"duration_days"`
	Speed        string  `json:"speed,omitempty"`
	DataCap      string  `json:"data_cap,omitempty"`
	Connection   string  `json:"connection_type,omitempty"`
	InstallFee   float64 `json:"installation_fee_iqd,omitempty"`
	Router       bool    `json:"router_included"`
	Active       bool    `json:"is_active"`
}

func toolSearchPlans() Tool {
	return Tool{
		Name: "search_service_plans",
		Description: "The internet plan catalogue for the provinces the caller's department can see: names, prices in IQD, " +
			"speeds, durations and what is included. THE ONLY source of any price or speed — you do not know a single plan price " +
			"until this returns one, and stating one you have not fetched is a serious error. " +
			"Use it for every question about باقة / باقات / اشتراك / سعر / أسعار / شكد سعر / أرخص / ارخص شي / سرعة / ميگا / إنترنت, " +
			"and for plans, packages, prices, speeds and 'what do you offer'. " +
			"Filters: free text, province, max price, minimum speed in Mbps; sorted cheapest first unless sort='speed'. " +
			"For 'the cheapest' just call it with no filters and read the first row; for 'cheap but at least 50 Mbps' set min_speed_mbps. " +
			"Returns at most 12 plans. If it returns none, say there are none rather than naming a price.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"query":{"type":"string","description":"text matched against plan, category and province names"},
				"province":{"type":"string","description":"province name (Arabic or English) to filter by"},
				"max_price":{"type":"number"},
				"min_speed_mbps":{"type":"number"},
				"sort":{"type":"string","enum":["price","speed"]},
				"include_inactive":{"type":"boolean","description":"default false"}
			},
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				Query           string  `json:"query"`
				Province        string  `json:"province"`
				MaxPrice        float64 `json:"max_price"`
				MinSpeedMbps    float64 `json:"min_speed_mbps"`
				Sort            string  `json:"sort"`
				IncludeInactive bool    `json:"include_inactive"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}

			provinces, err := visibleProvinces(ctx, d, actor)
			if err != nil {
				return nil, err
			}
			if in.Province != "" {
				var filtered []models.Province
				needle := strings.ToLower(strings.TrimSpace(in.Province))
				for _, p := range provinces {
					if strings.Contains(strings.ToLower(p.Name), needle) {
						filtered = append(filtered, p)
					}
				}
				if len(filtered) == 0 {
					return nil, Errf("no province matching %q is available to your department", in.Province)
				}
				provinces = filtered
			}

			query := strings.ToLower(strings.TrimSpace(in.Query))
			var plans []planView
			for _, prov := range provinces {
				cats, err := d.ServiceRepo.GetCategoriesByProvince(ctx, prov.ID)
				if err != nil {
					continue
				}
				for _, cat := range cats {
					if !in.IncludeInactive && !cat.IsActive {
						continue
					}
					rows, err := d.ServiceRepo.GetPlansByCategory(ctx, cat.ID)
					if err != nil {
						continue
					}
					for _, p := range rows {
						if !in.IncludeInactive && !p.IsActive {
							continue
						}
						if in.MaxPrice > 0 && p.Price > in.MaxPrice {
							continue
						}
						if in.MinSpeedMbps > 0 && speedMbps(p.Speed) < in.MinSpeedMbps {
							continue
						}
						if query != "" {
							hay := strings.ToLower(p.Name + " " + cat.Name + " " + prov.Name + " " + strOrEmpty(p.Speed) + " " + strOrEmpty(p.Description))
							if !strings.Contains(hay, query) {
								continue
							}
						}
						plans = append(plans, planView{
							PlanID:       p.ID.String(),
							Name:         p.Name,
							Category:     cat.Name,
							Province:     prov.Name,
							Price:        p.Price,
							DurationDays: p.DurationDays,
							Speed:        strOrEmpty(p.Speed),
							DataCap:      strOrEmpty(p.DataCap),
							Connection:   p.ConnectionType,
							InstallFee:   p.InstallationFee,
							Router:       p.RouterIncluded,
							Active:       p.IsActive,
						})
					}
				}
			}

			if in.Sort == "speed" {
				sort.Slice(plans, func(i, j int) bool {
					return speedMbps(&plans[i].Speed) > speedMbps(&plans[j].Speed)
				})
			} else {
				sort.Slice(plans, func(i, j int) bool { return plans[i].Price < plans[j].Price })
			}

			total := len(plans)
			if len(plans) > 12 {
				plans = plans[:12]
			}
			return map[string]any{
				"plans":       plans,
				"total_found": total,
				"truncated":   total > len(plans),
				"currency":    "IQD",
			}, nil
		},
	}
}

func toolGetPlanDetails() Tool {
	return Tool{
		Name:        "get_service_plan",
		Description: "Full details of one plan by plan_id (from search_service_plans), including description and cabinet notes.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{"plan_id":{"type":"string"}},
			"required":["plan_id"],
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				PlanID string `json:"plan_id"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			id, err := uuid.Parse(in.PlanID)
			if err != nil {
				return nil, Errf("invalid plan_id")
			}
			plan, err := d.ServiceRepo.GetPlanByID(ctx, id)
			if err != nil || plan == nil {
				return nil, Errf("plan not found")
			}
			cat, err := d.ServiceRepo.GetCategoryByID(ctx, plan.CategoryID)
			if err != nil || cat == nil {
				return nil, Errf("plan not found")
			}
			// Same visibility rule as the catalog endpoints after hardening.
			provinces, err := visibleProvinces(ctx, d, actor)
			if err != nil {
				return nil, err
			}
			visible := false
			var provinceName string
			for _, p := range provinces {
				if p.ID == cat.ProvinceID {
					visible = true
					provinceName = p.Name
					break
				}
			}
			if !visible {
				return nil, Errf("plan not found")
			}
			return map[string]any{
				"plan_id":       plan.ID.String(),
				"name":          plan.Name,
				"category":      cat.Name,
				"province":      provinceName,
				"price_iqd":     plan.Price,
				"duration_days": plan.DurationDays,
				"speed":         strOrEmpty(plan.Speed),
				"data_cap":      strOrEmpty(plan.DataCap),
				"connection":    plan.ConnectionType,
				"install_fee":   plan.InstallationFee,
				"router":        plan.RouterIncluded,
				"description":   clipRunes(strOrEmpty(plan.Description), 600),
				"cabinet_notes": clipRunes(strOrEmpty(plan.CabinetNotes), 400),
				"is_active":     plan.IsActive,
			}, nil
		},
	}
}
