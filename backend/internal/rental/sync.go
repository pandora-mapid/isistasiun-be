package rental

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const spaceKAIURL = "https://space-api.kai.id/api/v1/komersialasetram"

type spaceKAIResponse struct {
	Data []spaceKAIAsset `json:"data"`
}

type spaceKAIAsset struct {
	BlockID              string  `json:"blokid"`
	LocationName         string  `json:"namalokasi"`
	PlotName             string  `json:"namablok"`
	AreaName             string  `json:"namaarea"`
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	LandArea             float64 `json:"luastanah"`
	BuildingArea         float64 `json:"luasbangunan"`
	Rented               bool    `json:"rented"`
	CommercialValue      float64 `json:"nilaikomersial"`
	CommercialValueShown bool    `json:"nilaikomersialvis"`
	LastUpdated          string  `json:"lastupdated"`
}

// SyncSpaceKAI replaces the exact Manggarai snapshot in one transaction.
// Sudirman is intentionally not touched because the upstream API has no exact
// `namalokasi = Sudirman` records; its field-survey rows come from the seed.
func SyncSpaceKAI(ctx context.Context, db *pgxpool.Pool, client *http.Client) error {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spaceKAIURL, nil)
	if err != nil {
		return fmt.Errorf("build Space KAI request: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch Space KAI assets: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("fetch Space KAI assets: HTTP %d", res.StatusCode)
	}

	var payload spaceKAIResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode Space KAI assets: %w", err)
	}
	assets := make([]spaceKAIAsset, 0, len(payload.Data))
	for _, asset := range payload.Data {
		if strings.EqualFold(strings.TrimSpace(asset.LocationName), "manggarai") {
			assets = append(assets, asset)
		}
	}
	if len(assets) == 0 {
		return fmt.Errorf("Space KAI returned no exact Manggarai assets")
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rental sync: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after commit

	var stationID string
	if err := tx.QueryRow(ctx, `SELECT id FROM stations WHERE code = 'MRI'`).Scan(&stationID); err != nil {
		return fmt.Errorf("resolve Manggarai station: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM rental_assets WHERE station_id = $1 AND data_source = 'space_kai'`,
		stationID,
	); err != nil {
		return fmt.Errorf("clear old Space KAI snapshot: %w", err)
	}

	const insert = `
		INSERT INTO rental_assets (
			station_id, source_id, data_source, location_name, plot_name, area_name,
			latitude, longitude, land_area, building_area, rented, availability_status,
			commercial_value, commercial_value_visible, source_updated_at, note
		) VALUES ($1, $2, 'space_kai', $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	for _, asset := range assets {
		updated, err := time.Parse("2006-01-02 15:04:05", asset.LastUpdated)
		if err != nil {
			return fmt.Errorf("parse lastupdated for %s: %w", asset.BlockID, err)
		}
		status := "available"
		if asset.Rented {
			status = "occupied"
		}
		note := "Aset Space KAI; nilai komersial tidak ditampilkan oleh sumber."
		if _, err := tx.Exec(ctx, insert,
			stationID, asset.BlockID, asset.LocationName, asset.PlotName, asset.AreaName,
			asset.Latitude, asset.Longitude, asset.LandArea, asset.BuildingArea,
			asset.Rented, status, asset.CommercialValue, asset.CommercialValueShown,
			updated, note,
		); err != nil {
			return fmt.Errorf("insert Space KAI asset %s: %w", asset.BlockID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rental sync: %w", err)
	}
	return nil
}
