package models

type Question struct {
	ProviderId  string   `json:"providerId"`
	ExamID      string   `json:"examId"`
	QuestionID  string   `json:"questionId"`
	Options     []Option `json:"options"`
	Description []Item   `json:"description"`
	Answer      []Item   `json:"answer"`
}

type Option struct {
	Description []Item `json:"description"`
	IsCorrect   bool   `json:"isCorrect"`
	Id          string `json:"id"`
}

type Item struct {
	Content string `json:"content"`
	Type    string `json:"type"`
}
