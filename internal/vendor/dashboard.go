package vendor

import (
	"log"
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
		}
		whereClause := strings.Join(whereConditions, " AND ")

		query := `
WITH campaign_stats AS (
    SELECT
        v.vendor_id,
        v.vendor_name,
        l.campaign_id,
        c.campaign_name,
        COALESCE(cc.category_name, 'Unknown') AS vertical_name,
        COUNT(*)::int AS generated,
        (SUM(CASE WHEN l.status = 'ACCEPTED' THEN 1 ELSE 0 END))::int AS accepted,
        (SUM(CASE WHEN l.status = 'DUPLICATED' THEN 1 ELSE 0 END))::int AS duplicated,
        (SUM(CASE WHEN l.status = 'ERROR' THEN 1 ELSE 0 END))::int AS errored,
        COALESCE(SUM(l.revenue), 0)::float8 AS revenue,
        COALESCE(SUM(l.cost), 0)::float8 AS cost,
        (COALESCE(SUM(l.revenue), 0) - COALESCE(SUM(l.cost), 0))::float8 AS profit,
        (SUM(CASE WHEN LOWER(l.outcome_appointment_set) = 'yes' THEN 1 ELSE 0 END))::int AS appointments,
        (SUM(CASE WHEN LOWER(l.outcome_appointment_status) = 'sale' THEN 1 ELSE 0 END))::int AS sales,
        (SUM(CASE WHEN LOWER(l.outcome_return_status) = 'yes' THEN 1 ELSE 0 END))::int AS returned,
        (SUM(CASE WHEN LOWER(l.outcome_return_status) = 'yes' THEN COALESCE(l.revenue, 0) ELSE 0 END))::float8 AS returned_amount,
        COALESCE(SUM(l.outcome_sale_amount), 0)::float8 AS total_sale_amount,
        COALESCE(SUM(l.outcome_net_sale_amount), 0)::float8 AS total_net_sale_amount,
        (SUM(CASE WHEN LOWER(l.outcome_appointment_status) LIKE '%answering machine%' THEN 1 ELSE 0 END))::int AS answering_machine_count,
        (SUM(CASE WHEN LOWER(l.outcome_appointment_status) LIKE '%issue%' THEN 1 ELSE 0 END))::int AS issue_count,
        (SUM(CASE WHEN LOWER(l.outcome_appointment_status) LIKE '%hung up%' THEN 1 ELSE 0 END))::int AS hung_up_count,
        (SUM(CASE WHEN LOWER(l.outcome_appointment_status) LIKE '%pitch miss%' THEN 1 ELSE 0 END))::int AS pitch_miss_count,
        array_to_string(ARRAY_AGG(l.lead_id) FILTER (WHERE LOWER(l.outcome_appointment_status) = 'sale'), ',') AS sales_lead_ids,
        array_to_string(ARRAY_AGG(l.lead_id) FILTER (WHERE LOWER(l.outcome_return_status) = 'yes'), ',') AS returned_lead_ids,
        array_to_string(ARRAY_AGG(l.lead_id) FILTER (WHERE LOWER(l.outcome_appointment_set) = 'yes'), ',') AS appointments_lead_ids
    FROM vendor v
    INNER JOIN lead l ON v.vendor_id = l.vendor_id
    LEFT JOIN campaign c ON l.campaign_id = c.campaign_id
    LEFT JOIN campaign_category cc ON c.category_id = cc.category_id
    WHERE ` + whereClause + `
    GROUP BY v.vendor_id, v.vendor_name, l.campaign_id, c.campaign_name, cc.category_name
)
SELECT
    vendor_id,
    vendor_name,
    campaign_id,
    campaign_name,
    vertical_name,
    generated,
    accepted,
    duplicated,
    errored,
    revenue,
    cost,
    profit,
    appointments,
    sales,
    returned,
    returned_amount,
    total_sale_amount,
    total_net_sale_amount,
    answering_machine_count,
    issue_count,
    hung_up_count,
    pitch_miss_count,
    sales_lead_ids,
    returned_lead_ids,
    appointments_lead_ids
FROM campaign_stats
ORDER BY vendor_id, campaign_id
`

		rows, err := pool.Query(c.Context(), query, args...)
		if err != nil {
			log.Printf("vendor dashboard query error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal_error", "message": "failed to fetch vendor dashboard",
			})
		}
		defer rows.Close()

		vendorData := make(map[int64]*vendorEntry)
		for rows.Next() {
			var (
				vendorID, campaignID                                                                  int64
				vendorName, campaignName, verticalName                                                string
				generated, accepted, duplicated, errored, appointments, sales, returned               int
				answeringMachineCount, issueCount, hungUpCount, pitchMissCount                        int
				revenue, cost, profit, returnedAmount, totalSaleAmount, totalNetSaleAmount            float64
				salesLeadIDsStr, returnedLeadIDsStr, appointmentsLeadIDsStr                           string
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
				log.Printf("vendor dashboard scan error: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal_error", "message": "failed to fetch vendor dashboard",
				})
			}

			salesLeadIDs := splitNonEmpty(salesLeadIDsStr, ",")
			returnedLeadIDs := splitNonEmpty(returnedLeadIDsStr, ",")
			appointmentsLeadIDs := splitNonEmpty(appointmentsLeadIDsStr, ",")

			if _, ok := vendorData[vendorID]; !ok {
				empty := emptyStats()
				vendorData[vendorID] = &vendorEntry{
					VendorID:   vendorID,
					VendorName: vendorName,
					TotalStats: &empty,
					Campaigns:  []campaignEntry{},
				}
			}
			ent := vendorData[vendorID]

			stats := statsMap{
				Generated:              generated,
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
				TotalSaleAmount:        totalSaleAmount,
				TotalNetSaleAmount:     totalNetSaleAmount,
				AnsweringMachineCount: answeringMachineCount,
				IssueCount:            issueCount,
				HungUpCount:           hungUpCount,
				PitchMissCount:         pitchMissCount,
				SalesLeadIDs:          salesLeadIDs,
				ReturnedLeadIDs:       returnedLeadIDs,
				AppointmentsLeadIDs:   appointmentsLeadIDs,
			}
			ent.Campaigns = append(ent.Campaigns, campaignEntry{
				VendorID:      vendorID,
				CampaignID:    campaignID,
				CampaignName:  campaignName,
				VerticalName: verticalName,
				Stats:         stats,
			})
			addStats(ent.TotalStats, stats)
		}
		if err := rows.Err(); err != nil {
			log.Printf("vendor dashboard rows error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal_error", "message": "failed to fetch vendor dashboard",
			})
		}

		result := make([]fiber.Map, 0, len(vendorData))
		for _, v := range vendorData {
			campaigns := make([]fiber.Map, 0, len(v.Campaigns))
			for _, camp := range v.Campaigns {
				campaigns = append(campaigns, fiber.Map{
					"vendor_id":      camp.VendorID,
					"campaign_id":    camp.CampaignID,
					"campaign_name":  camp.CampaignName,
					"vertical_name":  camp.VerticalName,
					"stats":          statsToMap(camp.Stats),
				})
			}
			result = append(result, fiber.Map{
				"vendor_id":   v.VendorID,
				"vendor_name": v.VendorName,
				"totalStats":  statsToMap(*v.TotalStats),
				"campaigns":   campaigns,
			})
		}
		sortByRevenueDesc(result)
		return c.JSON(result)
	}
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
}

func statsToMap(s statsMap) fiber.Map {
	m := fiber.Map{
		"generated":                s.Generated,
		"accepted":                 s.Accepted,
		"duplicated":              s.Duplicated,
		"errored":                 s.Errored,
		"revenue":                 s.Revenue,
		"cost":                    s.Cost,
		"profit":                  s.Profit,
		"appointments":            s.Appointments,
		"sales":                   s.Sales,
		"returned":                s.Returned,
		"returned_amount":         s.ReturnedAmount,
		"total_sale_amount":       s.TotalSaleAmount,
		"total_net_sale_amount":    s.TotalNetSaleAmount,
		"answering_machine_count": s.AnsweringMachineCount,
		"issue_count":             s.IssueCount,
		"hung_up_count":           s.HungUpCount,
		"pitch_miss_count":        s.PitchMissCount,
	}
	if len(s.SalesLeadIDs) > 0 || len(s.ReturnedLeadIDs) > 0 || len(s.AppointmentsLeadIDs) > 0 {
		m["sales_lead_ids"] = s.SalesLeadIDs
		m["returned_lead_ids"] = s.ReturnedLeadIDs
		m["appointments_lead_ids"] = s.AppointmentsLeadIDs
	}
	return m
}

func sortByRevenueDesc(slice []fiber.Map) {
	for i := 0; i < len(slice); i++ {
		for j := i + 1; j < len(slice); j++ {
			totalI, _ := slice[i]["totalStats"].(fiber.Map)
			totalJ, _ := slice[j]["totalStats"].(fiber.Map)
			revI, _ := totalI["revenue"].(float64)
			revJ, _ := totalJ["revenue"].(float64)
			if revJ > revI {
				slice[i], slice[j] = slice[j], slice[i]
			}
		}
	}
}

type vendorEntry struct {
	VendorID   int64
	VendorName string
	TotalStats *statsMap
	Campaigns  []campaignEntry
}

type campaignEntry struct {
	VendorID      int64
	CampaignID    int64
	CampaignName  string
	VerticalName  string
	Stats         statsMap
}

type statsMap struct {
	Generated              int
	Accepted               int
	Duplicated             int
	Errored                int
	Revenue                float64
	Cost                   float64
	Profit                 float64
	Appointments           int
	Sales                  int
	Returned               int
	ReturnedAmount         float64
	TotalSaleAmount        float64
	TotalNetSaleAmount     float64
	AnsweringMachineCount int
	IssueCount             int
	HungUpCount            int
	PitchMissCount         int
	SalesLeadIDs           []string
	ReturnedLeadIDs        []string
	AppointmentsLeadIDs    []string
}
