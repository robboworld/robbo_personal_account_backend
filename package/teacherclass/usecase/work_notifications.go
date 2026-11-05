package usecase

import (
	"log"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// The bell for class work, Scratch and files alike. Like the roster rows, it is best effort: the
// hand-in or the review has already happened, so a failed notification is only logged.

// notifyWorkSubmitted: one row per assignment for each teacher of the class, the next hand-in refreshes it,
// or a class handing in at the end of a lesson would fill the bell.
func (u *UseCaseImpl) notifyWorkSubmitted(studentID string, a *models.ClassAssignmentDB) {
	inv, err := u.gw.GetInviteByCourseCohort(a.EdxCourseID, a.EdxCohortID)
	if err != nil {
		log.Printf("teacherclass: notify assignment %s owner: %v", a.ID, err)
		return
	}
	for _, row := range u.teacherNotifications(studentID, inv, "class_work_submitted", "class_work_submitted:"+a.ID) {
		row.Title = "Новая работа на проверке"
		row.Body = studentName(studentID) + ": «" + a.Title + "», класс «" + inv.DisplayName + "»"
		if err := u.notify.CreateOrUpdateByDedupe(row); err != nil {
			log.Printf("teacherclass: notify class %s teacher: %v", inv.ID, err)
		}
	}
}

// workReviewTitles: what the student hears about; the status "submitted" tells them nothing new.
var workReviewTitles = map[string]string{
	models.ReviewReturned:     "Работа возвращена на доработку",
	models.ReviewAccepted:     "Работа принята",
	models.ReviewShownInClass: "Работу показали на уроке",
}

// notifyWorkReviewed: one row per piece of work (a project copy or a file submission), the next
// review refreshes it.
func (u *UseCaseImpl) notifyWorkReviewed(studentID, workID, workTitle, status, comment string) {
	title, ok := workReviewTitles[status]
	if !ok {
		return
	}
	body := "«" + workTitle + "»"
	if comment != "" {
		body += ": " + comment
	}
	actionURL := "/myclasses"
	dedupeKey := "class_work_reviewed:" + workID
	row := &models.UserNotificationDB{
		RecipientUserID: studentID,
		Title:           title,
		Body:            body,
		Kind:            "class_work_reviewed",
		Severity:        "INFO",
		Source:          "system",
		ActionURL:       &actionURL,
		DedupeKey:       &dedupeKey,
	}
	if err := u.notify.CreateOrUpdateByDedupe(row); err != nil {
		log.Printf("teacherclass: notify work %s student: %v", workID, err)
	}
}

// notifyWorkEdited: one row per work for the student and one for each other teacher of the class;
// the next save refreshes them, and a teacher saves many times over one review.
func (u *UseCaseImpl) notifyWorkEdited(teacherID string, inv *models.ClassInviteDB, p *models.ScratchProjectDB, restored bool) {
	title, teachersTitle := "Учитель изменил вашу работу", "Учитель изменил работу ученика"
	if restored {
		title, teachersTitle = "Учитель отменил свои изменения в вашей работе", "Учитель отменил изменения в работе ученика"
	}
	actionURL := "/projects/" + p.ID
	dedupeKey := "class_work_edited:" + p.ID
	row := &models.UserNotificationDB{
		RecipientUserID: p.OwnerUserID,
		Title:           title,
		Body:            "«" + p.Title + "»",
		Kind:            "class_work_edited",
		Severity:        "INFO",
		Source:          "system",
		ActionURL:       &actionURL,
		DedupeKey:       &dedupeKey,
	}
	if err := u.notify.CreateOrUpdateByDedupe(row); err != nil {
		log.Printf("teacherclass: notify work %s student: %v", p.ID, err)
	}
	// Whoever saved knows: the board and the editor show it.
	for _, row := range u.teacherNotifications(teacherID, inv, "class_work_edited", "class_work_edited_teachers:"+p.ID) {
		row.Title = teachersTitle
		row.Body = studentName(p.OwnerUserID) + ": «" + p.Title + "», класс «" + inv.DisplayName + "»"
		if err := u.notify.CreateOrUpdateByDedupe(row); err != nil {
			log.Printf("teacherclass: notify class %s teacher: %v", inv.ID, err)
		}
	}
}
