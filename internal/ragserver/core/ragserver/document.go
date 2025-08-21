package ragserver

import (
	"strings"
)

type Document struct {
	FileID FileID
	Text   string
	Page   int
}

type Topic struct {
	Name     string
	Keywords []string
}

type RelevantTopics []Topic

func (rt RelevantTopics) IsRelevant(content string) (Topic, bool) {
	for len(rt) == 0 {
		return Topic{}, false
	}

	for _, topic := range rt {
		for _, keyword := range topic.Keywords {
			if strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				return topic, true
			}
		}
	}

	return Topic{}, false
}
