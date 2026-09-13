// Command ingestfield loads the field survey fixtures from the sibling
// `isistasiun-ai` repo into Postgres through the `/survey/*` endpoints.
//
// The fixtures (isistasiun-ai/data/source/field/) are the machine-readable
// form of the 2026-08-31/09-01 tally-counter survey. Until this ran, no row of
// it existed in `flow_observations` or `entry_conversion_observations`, so the
// Monte Carlo pipeline had nothing real to read.
//
//	make be-ingest-field              # submit
//	make be-ingest-field DRY_RUN=1    # print what would be submitted
//
// Safe to re-run: every submission carries an Idempotency-Key derived from the
// fixture row id, so replays return the original row instead of duplicating.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/list-pandora/isi-stasiun-backend/internal/config"
	"github.com/list-pandora/isi-stasiun-backend/internal/database"
)

func main() {
	dir := flag.String("dir", "../../isistasiun-ai/data/source/field", "direktori fixture lapangan")
	apiBase := flag.String("api", "http://localhost:8080/api/v1", "base URL API")
	surveyorID := flag.String("surveyor", "tim-survei-list-pandora", "surveyor_id untuk semua baris")
	dryRun := flag.Bool("dry-run", false, "cetak payload tanpa mengirim")
	flag.Parse()

	cfg := config.Load()
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("gagal konek database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stations, err := loadStations(ctx, db)
	if err != nil {
		log.Fatalf("gagal membaca tabel stations: %v", err)
	}
	entranceIDs, err := ensureFieldEntrances(ctx, db, stations, *dryRun)
	if err != nil {
		log.Fatalf("gagal menyiapkan entrance lapangan: %v", err)
	}

	var entries entryFixture
	if err := readFixture(filepath.Join(*dir, "entry-conversion.json"), &entries); err != nil {
		log.Fatalf("%v", err)
	}
	var flows flowFixture
	if err := readFixture(filepath.Join(*dir, "flow-observations.json"), &flows); err != nil {
		log.Fatalf("%v", err)
	}

	entrySubs, entrySkips, err := buildEntryConversions(entries.Data, stations, *surveyorID)
	if err != nil {
		log.Fatalf("transform entry-conversion: %v", err)
	}
	flowSubs, flowSkips, err := buildFlowObservations(flows.Data, stations, entranceIDs, *surveyorID)
	if err != nil {
		log.Fatalf("transform flow-observations: %v", err)
	}

	for _, s := range append(append([]skip{}, flowSkips...), entrySkips...) {
		log.Printf("lewati %s: %s", s.ID, s.Reason)
	}

	subs := append(flowSubs, entrySubs...)
	if *dryRun {
		for _, sub := range subs {
			body, _ := json.Marshal(sub.Body)
			log.Printf("POST %s [%s] %s", sub.Path, sub.IdempotencyKey, body)
		}
		log.Printf("dry-run: %d baris siap dikirim, %d dilewati", len(subs), len(entrySkips)+len(flowSkips))
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	sent := 0
	for _, sub := range subs {
		if err := post(ctx, client, *apiBase, cfg.PipelineServiceAPIKey, sub); err != nil {
			log.Fatalf("kirim %s: %v", sub.IdempotencyKey, err)
		}
		sent++
	}
	log.Printf("selesai: %d baris terkirim (%d arus, %d entry-conversion), %d dilewati",
		sent, len(flowSubs), len(entrySubs), len(entrySkips)+len(flowSkips))
}

func readFixture(path string, into any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("baca %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// loadStations keys the two study stations by the name the fixtures use.
func loadStations(ctx context.Context, db *pgxpool.Pool) (map[string]station, error) {
	rows, err := db.Query(ctx, `SELECT id, name, code FROM stations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]station{}
	for rows.Next() {
		var id, name, code string
		if err := rows.Scan(&id, &name, &code); err != nil {
			return nil, err
		}
		out[name] = station{ID: id, Code: code}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("tabel stations kosong — jalankan seed stasiun dulu")
	}
	return out, nil
}

// ensureFieldEntrances inserts the surveyed doors if they are not there yet and
// returns their ids keyed by "<station code>:<fixture pintu label>".
//
// `/survey/flow-observations` needs an existing `station_entrances` row
// (foreign key, migration 000003) and there is no write endpoint for entrances
// — that table belongs to the stations owner. This only inserts rows, never
// touches their schema or code, and yields to whatever is already there.
//
// Under -dry-run nothing is written: existing ids are read back, and doors that
// do not exist yet are reported as such rather than created.
func ensureFieldEntrances(ctx context.Context, db *pgxpool.Pool, stations map[string]station, dryRun bool) (map[string]string, error) {
	byCode := map[string]station{}
	for _, st := range stations {
		byCode[st.Code] = st
	}

	const upsert = `
		INSERT INTO station_entrances (station_id, label, location)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography)
		ON CONFLICT (station_id, label) DO UPDATE SET location = EXCLUDED.location
		RETURNING id
	`
	const lookup = `SELECT id FROM station_entrances WHERE station_id = $1 AND label = $2`

	out := map[string]string{}
	for _, spec := range fieldEntrances {
		st, ok := byCode[spec.StationCode]
		if !ok {
			log.Printf("lewati entrance %s/%s: stasiun %s tidak ada di DB", spec.StationCode, spec.Label, spec.StationCode)
			continue
		}

		var id string
		var err error
		if dryRun {
			err = db.QueryRow(ctx, lookup, st.ID, spec.Label).Scan(&id)
			if err != nil {
				log.Printf("dry-run: entrance %s/%s belum ada, akan dibuat saat ingest", spec.StationCode, spec.Label)
				id = "(entrance baru: " + spec.Label + ")"
				err = nil
			}
		} else {
			err = db.QueryRow(ctx, upsert, st.ID, spec.Label, spec.Lon, spec.Lat).Scan(&id)
		}
		if err != nil {
			return nil, fmt.Errorf("entrance %s/%s: %w", spec.StationCode, spec.Label, err)
		}
		out[spec.StationCode+":"+spec.Pintu] = id
	}
	return out, nil
}

func post(ctx context.Context, client *http.Client, apiBase, serviceKey string, sub submission) error {
	payload, err := json.Marshal(sub.Body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+sub.Path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", serviceKey)
	req.Header.Set("Idempotency-Key", sub.IdempotencyKey)

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("HTTP %d: %s", res.StatusCode, body)
	}
	return nil
}
