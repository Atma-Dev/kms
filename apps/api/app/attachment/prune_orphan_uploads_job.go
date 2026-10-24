package attachment

import (
	"api-kms/config"
	"context"
	"fmt"
	"time"
)

const PruneOrphanUploadsJobName = "prune-orphan-uploads"

// PruneOrphanUploadsJob reclaims inline images that never reached a document.
//
// FR-8 uploads an image the moment it is dropped into the editor, against the
// space rather than the document — the editor needs a URL before there is a
// document to attach it to. Every draft abandoned after an image was dropped
// therefore leaves a file nothing points at, and nothing else ever removes it:
// a document delete cascades to its own attachments, but these belong to no
// document to begin with.
type PruneOrphanUploadsJob struct {
	repository *Repository
}

func NewPruneOrphanUploadsJob(repository *Repository) *PruneOrphanUploadsJob {
	return &PruneOrphanUploadsJob{repository: repository}
}

func (j *PruneOrphanUploadsJob) Name() string { return PruneOrphanUploadsJobName }

// DefaultCron runs hourly, not daily. The retention window is measured in
// hours because it only has to outlive a writing session, so a daily sweep
// would leave a morning's abandoned uploads sitting until the small hours.
func (j *PruneOrphanUploadsJob) DefaultCron() string { return "15 * * * *" }

func (j *PruneOrphanUploadsJob) Run(ctx context.Context) (string, error) {
	retention := time.Duration(config.AppConfig.UploadOrphanRetentionHours) * time.Hour
	cutoff := time.Now().Add(-retention)

	orphans, err := j.repository.OrphanedInline(ctx, cutoff)
	if err != nil {
		return "", err
	}

	var removed, failed int
	for i := range orphans {
		// File first, then the row — the same order Service.DeleteById avoids,
		// and for the opposite reason. There the row is what makes the file
		// reachable, so losing it first is recoverable; here nothing points at
		// the file already, so dropping the row first would lose the only
		// record of which file to delete.
		if err := remove(orphans[i].FilePath); err != nil {
			failed++
			continue
		}
		if err := j.repository.DeleteById(ctx, orphans[i].ID.String()); err != nil {
			failed++
			continue
		}
		removed++
	}

	// Failures are counted, not returned: one unreadable file must not stop
	// the sweep, and the run is still a success — the next hour retries it.
	if failed > 0 {
		return fmt.Sprintf("%d orphaned upload(s) removed, %d failed", removed, failed), nil
	}
	return fmt.Sprintf("%d orphaned upload(s) removed", removed), nil
}
