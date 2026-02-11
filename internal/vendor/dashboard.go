package vendor

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Dashboard returns a fiber.Handler that serves GET /vendor/dashboard with optional start_date and end_date query params.
// Uses raw SQL over vendor, lead, campaign, campaign_category. Requires JWT when mounted.
func Dashboard(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if pool == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "service_unavailable", "message": "database not configured",
			})
		}

		var startDate, endDate *time.Time
		if s := c.Query("start_date"); s != "" {
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "validation_error", "message": "invalid start_date; use YYYY-MM-DD",
				})
			}
			startDate = &t
		}
		if s := c.Query("end_date"); s != "" {
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "validation_error", "message": "invalid end_date; use YYYY-MM-DD",
				})
			}
			endDate = &t
		}

		result, err := fetchDashboardData(c.Context(), pool, startDate, endDate)
		if err != nil {
			log.Printf("vendor dashboard error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal_error", "message": "failed to fetch vendor dashboard",
			})
		}

		return c.JSON(result)
	}
}

func fetchDashboardData(ctx context.Context, pool *pgxpool.Pool, startDate, endDate *time.Time) ([]vendorDashboardItem, error) {
	whereConditions := []string{"v.is_active = true"}
	var args []interface{}
	argNum := 1

	if startDate != nil {
		startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
		whereConditions = append(whereConditions, "l.lead_creation_date >= $"+strconv.Itoa(argNum))
		args = append(args, startOfDay)
		argNum++
	}
	if endDate != nil {
		nextDay := endDate.AddDate(0, 0, 1)
		endExclusive := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, nextDay.Location())
		whereConditions = append(whereConditions, "l.lead_creation_date < $"+strconv.Itoa(argNum))
		args = append(args, endExclusive)
		argNum++
	}
	whereClause := strings.Join(whereConditions, " AND ")

	// Optimized query: removed CTE, use COUNT FILTER, simplified aggregations
	query := `
SELECT
    v.vendor_id,
    v.vendor_name,
    l.campaign_id,
    COALESCE(c.campaign_name, 'Unknown') AS campaign_name,
    COALESCE(cc.category_name, 'Unknown') AS vertical_name,
    COUNT(*)::int AS generated,
    COUNT(*) FILTER (WHERE l.status = 'ACCEPTED')::int AS accepted,
    COUNT(*) FILTER (WHERE l.status = 'DUPLICATED')::int AS duplicated,
    COUNT(*) FILTER (WHERE l.status = 'ERROR')::int AS errored,
    COALESCE(SUM(l.revenue), 0)::float8 AS revenue,
    COALESCE(SUM(l.cost), 0)::float8 AS cost,
    (COALESCE(SUM(l.revenue), 0) - COALESCE(SUM(l.cost), 0))::float8 AS profit,
    COUNT(*) FILTER (WHERE LOWER(l.outcome_appointment_set) = 'yes')::int AS appointments,
    COUNT(*) FILTER (WHERE LOWER(l.outcome_appointment_status) = 'sale')::int AS sales,
    COUNT(*) FILTER (WHERE LOWER(l.outcome_return_status) = 'yes')::int AS returned,
    COALESCE(SUM(l.revenue) FILTER (WHERE LOWER(l.outcome_return_status) = 'yes'), 0)::float8 AS returned_amount,
    COALESCE(SUM(l.outcome_sale_amount), 0)::float8 AS total_sale_amount,
    COALESCE(SUM(l.outcome_net_sale_amount), 0)::float8 AS total_net_sale_amount,
    COUNT(*) FILTER (WHERE l.outcome_appointment_status ILIKE '%answering machine%')::int AS answering_machine_count,
    COUNT(*) FILTER (WHERE l.outcome_appointment_status ILIKE '%issue%')::int AS issue_count,
    COUNT(*) FILTER (WHERE l.outcome_appointment_status ILIKE '%hung up%')::int AS hung_up_count,
    COUNT(*) FILTER (WHERE l.outcome_appointment_status ILIKE '%pitch miss%')::int AS pitch_miss_count,
    array_to_string(ARRAY_AGG(l.lead_id) FILTER (WHERE LOWER(l.outcome_appointment_status) = 'sale'), ',') AS sales_lead_ids,
    array_to_string(ARRAY_AGG(l.lead_id) FILTER (WHERE LOWER(l.outcome_return_status) = 'yes'), ',') AS returned_lead_ids,
    array_to_string(ARRAY_AGG(l.lead_id) FILTER (WHERE LOWER(l.outcome_appointment_set) = 'yes'), ',') AS appointments_lead_ids
FROM vendor v
INNER JOIN lead l ON v.vendor_id = l.vendor_id
LEFT JOIN campaign c ON l.campaign_id = c.campaign_id
LEFT JOIN campaign_category cc ON c.category_id = cc.category_id
WHERE ` + whereClause + `
GROUP BY v.vendor_id, v.vendor_name, l.campaign_id, c.campaign_name, cc.category_name
ORDER BY v.vendor_id, l.campaign_id
`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vendorData := make(map[int64]*vendorEntry, 50)

	for rows.Next() {
		var (
			vendorID, campaignID                                                       int64
			vendorName, campaignName, verticalName                                     string
			generated, accepted, duplicated, errored, appointments, sales, returned    int
			answeringMachineCount, issueCount, hungUpCount, pitchMissCount             int
			revenue, cost, profit, returnedAmount, totalSaleAmount, totalNetSaleAmount float64
			salesLeadIDsStr, returnedLeadIDsStr, appointmentsLeadIDsStr                *string
		)
		err := rows.Scan(
			&vendorID, &vendorName, &campaignID, &campaignName, &verticalName,
			&generated, &accepted, &duplicated, &errored,
			&revenue, &cost, &profit,
			&appointments, &sales, &returned, &returnedAmount,
			&totalSaleAmount, &totalNetSaleAmount,
			&answeringMachineCount, &issueCount, &hungUpCount, &pitchMissCount,
			&salesLeadIDsStr, &returnedLeadIDsStr, &appointmentsLeadIDsStr,
		)
		if err != nil {
			return nil, err
		}

		// Initialize vendor entry if needed
		if _, ok := vendorData[vendorID]; !ok {
			empty := emptyStats()
			vendorData[vendorID] = &vendorEntry{
				VendorID:   vendorID,
				VendorName: vendorName,
				TotalStats: &empty,
				Campaigns:  make([]campaignEntry, 0, 10),
			}
		}
		ent := vendorData[vendorID]

		// Build stats
		stats := statsMap{
			Generated:             generated,
			Accepted:              accepted,
			Duplicated:            duplicated,
			Errored:               errored,
			Revenue:               revenue,
			Cost:                  cost,
			Profit:                profit,
			Appointments:          appointments,
			Sales:                 sales,
			Returned:              returned,
			ReturnedAmount:        returnedAmount,
			TotalSaleAmount:       totalSaleAmount,
			TotalNetSaleAmount:    totalNetSaleAmount,
			AnsweringMachineCount: answeringMachineCount,
			IssueCount:            issueCount,
			HungUpCount:           hungUpCount,
			PitchMissCount:        pitchMissCount,
			SalesLeadIDs:          splitNonEmpty(ptrStr(salesLeadIDsStr), ","),
			ReturnedLeadIDs:       splitNonEmpty(ptrStr(returnedLeadIDsStr), ","),
			AppointmentsLeadIDs:   splitNonEmpty(ptrStr(appointmentsLeadIDsStr), ","),
		}

		// Add campaign
		ent.Campaigns = append(ent.Campaigns, campaignEntry{
			VendorID:     vendorID,
			CampaignID:   campaignID,
			CampaignName: campaignName,
			VerticalName: verticalName,
			Stats:        stats,
		})

		// Aggregate to vendor totals
		addStats(ent.TotalStats, stats)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build result
	result := make([]vendorDashboardItem, 0, len(vendorData))
	for _, v := range vendorData {
		campaigns := make([]campaignItemDTO, len(v.Campaigns))
		for i, camp := range v.Campaigns {
			campaigns[i] = campaignItemDTO{
				VendorID:     camp.VendorID,
				CampaignID:   camp.CampaignID,
				CampaignName: camp.CampaignName,
				VerticalName: camp.VerticalName,
				Stats:        toCampaignStatsDTO(camp.Stats),
			}
		}
		result = append(result, vendorDashboardItem{
			VendorName: v.VendorName,
			VendorID:   v.VendorID,
			TotalStats: toTotalStatsDTO(*v.TotalStats),
			Campaigns:  campaigns,
		})
	}

	// Sort by revenue descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalStats.Revenue > result[j].TotalStats.Revenue
	})

	return result, nil
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func splitNonEmpty(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func emptyStats() statsMap {
	return statsMap{
		SalesLeadIDs:        []string{},
		ReturnedLeadIDs:     []string{},
		AppointmentsLeadIDs: []string{},
	}
}

func addStats(dst *statsMap, src statsMap) {
	dst.Generated += src.Generated
	dst.Accepted += src.Accepted
	dst.Duplicated += src.Duplicated
	dst.Errored += src.Errored
	dst.Revenue += src.Revenue
	dst.Cost += src.Cost
	dst.Profit += src.Profit
	dst.Appointments += src.Appointments
	dst.Sales += src.Sales
	dst.Returned += src.Returned
	dst.ReturnedAmount += src.ReturnedAmount
	dst.TotalSaleAmount += src.TotalSaleAmount
	dst.TotalNetSaleAmount += src.TotalNetSaleAmount
	dst.AnsweringMachineCount += src.AnsweringMachineCount
	dst.IssueCount += src.IssueCount
	dst.HungUpCount += src.HungUpCount
	dst.PitchMissCount += src.PitchMissCount
	// Note: Lead IDs are not aggregated across campaigns
}

func toTotalStatsDTO(s statsMap) totalStatsDTO {
	return totalStatsDTO{
		Generated:             s.Generated,
		Accepted:              s.Accepted,
		Duplicated:            s.Duplicated,
		Errored:               s.Errored,
		Revenue:               s.Revenue,
		Cost:                  s.Cost,
		Profit:                s.Profit,
		Appointments:          s.Appointments,
		Sales:                 s.Sales,
		Returned:              s.Returned,
		ReturnedAmount:        s.ReturnedAmount,
		TotalSaleAmount:       s.TotalSaleAmount,
		TotalNetSaleAmount:    s.TotalNetSaleAmount,
		AnsweringMachineCount: s.AnsweringMachineCount,
		IssueCount:            s.IssueCount,
		HungUpCount:           s.HungUpCount,
		PitchMissCount:        s.PitchMissCount,
	}
}

func toCampaignStatsDTO(s statsMap) campaignStatsDTO {
	return campaignStatsDTO{
		totalStatsDTO:       toTotalStatsDTO(s),
		SalesLeadIDs:        s.SalesLeadIDs,
		ReturnedLeadIDs:     s.ReturnedLeadIDs,
		AppointmentsLeadIDs: s.AppointmentsLeadIDs,
	}
}

type vendorEntry struct {
	VendorID   int64
	VendorName string
	TotalStats *statsMap
	Campaigns  []campaignEntry
}

type campaignEntry struct {
	VendorID     int64
	CampaignID   int64
	CampaignName string
	VerticalName string
	Stats        statsMap
}

type statsMap struct {
	Generated             int
	Accepted              int
	Duplicated            int
	Errored               int
	Revenue               float64
	Cost                  float64
	Profit                float64
	Appointments          int
	Sales                 int
	Returned              int
	ReturnedAmount        float64
	TotalSaleAmount       float64
	TotalNetSaleAmount    float64
	AnsweringMachineCount int
	IssueCount            int
	HungUpCount           int
	PitchMissCount        int
	SalesLeadIDs          []string
	ReturnedLeadIDs       []string
	AppointmentsLeadIDs   []string
}

// Response DTOs (field order matches Python JSON output)
type vendorDashboardItem struct {
	VendorName string            `json:"vendor_name"`
	VendorID   int64             `json:"vendor_id"`
	TotalStats totalStatsDTO     `json:"totalStats"`
	Campaigns  []campaignItemDTO `json:"campaigns"`
}

type totalStatsDTO struct {
	Generated             int     `json:"generated"`
	Accepted              int     `json:"accepted"`
	Duplicated            int     `json:"duplicated"`
	Errored               int     `json:"errored"`
	Revenue               float64 `json:"revenue"`
	Cost                  float64 `json:"cost"`
	Profit                float64 `json:"profit"`
	Appointments          int     `json:"appointments"`
	Sales                 int     `json:"sales"`
	Returned              int     `json:"returned"`
	ReturnedAmount        float64 `json:"returned_amount"`
	TotalSaleAmount       float64 `json:"total_sale_amount"`
	TotalNetSaleAmount    float64 `json:"total_net_sale_amount"`
	AnsweringMachineCount int     `json:"answering_machine_count"`
	IssueCount            int     `json:"issue_count"`
	HungUpCount           int     `json:"hung_up_count"`
	PitchMissCount        int     `json:"pitch_miss_count"`
}

type campaignItemDTO struct {
	VendorID     int64            `json:"vendor_id"`
	CampaignID   int64            `json:"campaign_id"`
	CampaignName string           `json:"campaign_name"`
	VerticalName string           `json:"vertical_name"`
	Stats        campaignStatsDTO `json:"stats"`
}

type campaignStatsDTO struct {
	totalStatsDTO
	SalesLeadIDs        []string `json:"sales_lead_ids,omitempty"`
	ReturnedLeadIDs     []string `json:"returned_lead_ids,omitempty"`
	AppointmentsLeadIDs []string `json:"appointments_lead_ids,omitempty"`
}
