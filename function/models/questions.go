package models

type Question struct {
	ProviderId   string        `json:"providerId"`
	ExamID       string        `json:"examId"`
	QuestionID   string        `json:"questionId"`
	Options      []Option      `json:"options"`
	Descriptions []Description `json:"description"` // maybe rename description -> descriptions and reflect the change on db schema
}

type Option struct {
	Descriptions []Description `json:"description"` // maybe rename description -> descriptions and reflect the change on db schema
	IsCorrect    bool          `json:"isCorrect"`
	Id           string        `json:"id"`
}

type Description struct {
	Content string `json:"content"`
	Type    string `json:"type"`
}
