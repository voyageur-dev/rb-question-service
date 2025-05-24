package models

type GetQuestionsResponse struct {
	Questions []Question `json:"questions"`
}

type Question struct {
	ExamID      string   `json:"examId"`
	QuestionID  int      `json:"questionId"`
	Options     []Option `json:"options"`
	Question    string   `json:"question"`
	S3ImageURLs []string `json:"s3ImageUrls"`
}

type Option struct {
	IsCorrect   bool     `json:"isCorrect"`
	Text        string   `json:"text"`
	S3ImageURLs []string `json:"s3ImageUrls"`
}
