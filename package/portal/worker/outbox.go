package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	"github.com/spf13/viper"
)

type OutboxWorker struct {
	portal portalgateway.Gateway
	edx    edx.UseCase
	lms    *lmsdb.Reader
}

func NewOutboxWorker(portal portalgateway.Gateway, edxUC edx.UseCase) *OutboxWorker {
	w := &OutboxWorker{portal: portal, edx: edxUC}
	if reader, err := lmsdb.NewReaderFromConfig(); err == nil {
		w.lms = reader
	} else {
		log.Printf("[outbox] lms mysql reader disabled: %v", err)
	}
	return w
}

func (w *OutboxWorker) RunOnce(limit int) {
	rows, err := w.portal.FetchPendingOutbox(limit)
	if err != nil {
		log.Printf("[outbox] fetch: %v", err)
		return
	}
	for _, row := range rows {
		if err := w.process(row.ID, row.EventType, row.Payload); err != nil {
			_ = w.portal.MarkOutboxFailed(row.ID, err.Error())
			log.Printf("[outbox] id=%d failed: %v", row.ID, err)
			continue
		}
		_ = w.portal.MarkOutboxDone(row.ID)
	}
}

func (w *OutboxWorker) RunLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[outbox] stopped: %v", ctx.Err())
			return
		case <-ticker.C:
			if viper.GetBool("portalOutbox.enabled") {
				w.RunOnce(20)
			}
		}
	}
}

func (w *OutboxWorker) process(id int64, eventType, payload string) error {
	switch eventType {
	case "enrollment.requested":
		return w.processEnrollment(payload)
	case "user.link_requested":
		return w.processUserLink(payload)
	default:
		return fmt.Errorf("unknown event_type %s", eventType)
	}
}

type enrollmentPayload struct {
	UserID   string `json:"userId"`
	CourseID string `json:"courseId"`
	Username string `json:"username"`
}

func (w *OutboxWorker) processEnrollment(payload string) error {
	var p enrollmentPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	uid, err := strconv.ParseInt(p.UserID, 10, 64)
	if err != nil {
		return err
	}
	if w.lms != nil {
		has, err := w.lms.HasEnrollment(uid, p.CourseID)
		if err != nil {
			return err
		}
		if has {
			return nil
		}
	}
	if w.edx == nil {
		return fmt.Errorf("edx use case not configured")
	}
	params := map[string]interface{}{
		"course_details": map[string]interface{}{"course_id": p.CourseID},
		"user":           p.Username,
	}
	_, err = w.edx.PostWithAuth(viper.GetString("api_urls.postEnrollment"), params)
	return err
}

type userLinkPayload struct {
	Email string `json:"email"`
}

func (w *OutboxWorker) processUserLink(payload string) error {
	var p userLinkPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	if w.lms == nil {
		return fmt.Errorf("lms reader required for user.link_requested")
	}
	u, err := w.lms.LookupAuthUserByEmail(p.Email)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("auth_user not found for %s", p.Email)
	}
	edxID := strconv.FormatInt(u.ID, 10)
	_, err = w.portal.UpsertUserLinkByOIDC("", p.Email, u.Username, edxID)
	return err
}
