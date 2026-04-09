// Package retention enforces backup job lifetime policies.
// Supports both simple max-age (Days) and Grandfather-Father-Son (GFS) rotation.
package retention

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smcsoluciones/backup-system/agent/internal/config"
	"github.com/smcsoluciones/backup-system/agent/internal/destination"
)

type runEntry struct {
	jobID string
	ts    string
	t     time.Time
	objs  []string
}

// Apply lists all job runs in the destination and deletes those that fall
// outside the configured retention policy.
//
//   - If cfg.GFS.Enabled is true, uses Grandfather-Father-Son rotation
//     (daily / weekly / monthly windows).
//   - Otherwise, if cfg.Days > 0, uses a simple max-age cutoff.
//   - If neither is set, Apply is a no-op (keep everything).
func Apply(dest destination.Writer, cfg config.RetentionConfig, log *zap.Logger) error {
	all, err := dest.List("jobs/")
	if err != nil {
		return fmt.Errorf("retention: list objects: %w", err)
	}
	if len(all) == 0 {
		return nil
	}

	// Group objects by run: jobs/{jobID}/{ts}/...
	type runKey struct{ jobID, ts string }
	grouped := make(map[runKey][]string)
	for _, obj := range all {
		parts := strings.SplitN(obj, "/", 4)
		if len(parts) < 4 {
			continue
		}
		k := runKey{jobID: parts[1], ts: parts[2]}
		grouped[k] = append(grouped[k], obj)
	}

	// Parse timestamps into sortable entries.
	var runs []runEntry
	for k, objs := range grouped {
		t, err := time.Parse("20060102T150405Z", k.ts)
		if err != nil {
			log.Warn("retention: unparseable run timestamp, skipping",
				zap.String("job_id", k.jobID), zap.String("ts", k.ts))
			continue
		}
		runs = append(runs, runEntry{jobID: k.jobID, ts: k.ts, t: t, objs: objs})
	}

	if len(runs) == 0 {
		return nil
	}

	// Sort newest-first so GFS selection always picks the most recent representative.
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].t.After(runs[j].t)
	})

	// Determine which run indices to keep.
	keep := make(map[int]bool)
	switch {
	case cfg.GFS.Enabled:
		keep = selectGFS(runs, cfg.GFS)
	case cfg.Days > 0:
		cutoff := time.Now().UTC().AddDate(0, 0, -cfg.Days)
		for i, r := range runs {
			if r.t.After(cutoff) {
				keep[i] = true
			}
		}
	default:
		return nil // no retention policy configured
	}

	var deleted, errors int
	for i, r := range runs {
		if keep[i] {
			continue
		}
		log.Info("retention: deleting expired run",
			zap.String("job_id", r.jobID),
			zap.String("ts", r.ts),
			zap.Time("run_time", r.t),
			zap.Int("objects", len(r.objs)),
		)
		for _, obj := range r.objs {
			if delErr := dest.Delete(obj); delErr != nil {
				log.Warn("retention: delete object failed",
					zap.String("obj", obj), zap.Error(delErr))
				errors++
			} else {
				deleted++
			}
		}
	}

	log.Info("retention: cleanup complete",
		zap.Int("runs_total", len(runs)),
		zap.Int("runs_kept", len(keep)),
		zap.Int("runs_deleted", len(runs)-len(keep)),
		zap.Int("objects_deleted", deleted),
		zap.Int("delete_errors", errors),
	)
	return nil
}

// selectGFS returns the set of run indices (in a newest-first sorted slice)
// to keep under the Grandfather-Father-Son policy.
//
// Strategy:
//   - Daily  : keep one backup per calendar day  for the most recent KeepDaily  entries
//   - Weekly : keep one backup per ISO week       for the most recent KeepWeekly entries
//   - Monthly: keep one backup per calendar month for the most recent KeepMonthly entries
//
// An index may satisfy more than one tier; it is still stored only once.
func selectGFS(runs []runEntry, cfg config.GFSConfig) map[int]bool {
	keep := make(map[int]bool)

	// Daily tier
	seenDay := make(map[string]bool)
	dailyKept := 0
	for i, r := range runs {
		if dailyKept >= cfg.KeepDaily {
			break
		}
		day := r.t.UTC().Format("2006-01-02")
		if !seenDay[day] {
			seenDay[day] = true
			keep[i] = true
			dailyKept++
		}
	}

	// Weekly tier
	seenWeek := make(map[string]bool)
	weeklyKept := 0
	for i, r := range runs {
		if weeklyKept >= cfg.KeepWeekly {
			break
		}
		year, week := r.t.UTC().ISOWeek()
		wk := fmt.Sprintf("%04d-W%02d", year, week)
		if !seenWeek[wk] {
			seenWeek[wk] = true
			keep[i] = true
			weeklyKept++
		}
	}

	// Monthly tier
	seenMonth := make(map[string]bool)
	monthlyKept := 0
	for i, r := range runs {
		if monthlyKept >= cfg.KeepMonthly {
			break
		}
		month := r.t.UTC().Format("2006-01")
		if !seenMonth[month] {
			seenMonth[month] = true
			keep[i] = true
			monthlyKept++
		}
	}

	return keep
}
