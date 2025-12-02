package model

// CheckLinksRequest - запрос на проверку ссылок
type CheckLinksRequest struct {
	Links []string `json:"links"`
}

// CheckLinksResponse - ответ с результатами проверки
type CheckLinksResponse struct {
	Links    map[string]string `json:"links"`
	LinksNum int64             `json:"links_num"`
}

// GetReportRequest - запрос на получение отчёта
type GetReportRequest struct {
	LinksList []int64 `json:"links_list"`
}

// LinkSet - набор ссылок с результатами проверки
type LinkSet struct {
	ID    int64             `json:"id"`
	Links map[string]string `json:"links"`
}

// PendingTask - задача, ожидающая обработки
type PendingTask struct {
	ID    int64    `json:"id"`
	Links []string `json:"links"`
}
