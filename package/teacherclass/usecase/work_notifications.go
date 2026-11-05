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
