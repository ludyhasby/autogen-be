package entity

import (
	helperconverter "logisfy/helper/converter"
	modelresponse "logisfy/internal/model/response"
	"time"
)

type NewsEntity struct {
	NewsID      uint64    `gorm:"primaryKey;autoIncrement" json:"news_id"`
	Source      string    `gorm:"not null"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"null"`
	Url         string    `gorm:"not null"`
	UrlToImage  string    `gorm:"null"`
	PublishedAt time.Time `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (t NewsEntity) TableName() string {
	return "news"
}

func (t NewsEntity) CheckFound() bool {
	return t.NewsID > 0
}

func (t NewsEntity) ConvertToList(entities []NewsEntity) (resp []modelresponse.ListNews) {
	for _, entity := range entities {
		publishedAtStr := helperconverter.ConvertTimeToString(&entity.PublishedAt)
		resp = append(resp, modelresponse.ListNews{
			Source:      entity.Source,
			Title:       entity.Title,
			Description: entity.Description,
			Url:         entity.Url,
			UrlToImage:  entity.UrlToImage,
			PublishedAt: publishedAtStr,
		})
	}
	return
}

func (t NewsEntity) ConvertToEntity(respFetch modelresponse.NewsFetchResponse, entitiesExisting []NewsEntity) (entities []*NewsEntity) {
	existingTitles := make(map[string]struct{}, len(entitiesExisting))
	for _, existing := range entitiesExisting {
		existingTitles[existing.Title] = struct{}{}
	}

	for _, article := range respFetch.Articles {
		publishedAt, err := time.Parse(time.RFC3339, article.PublishedAt)
		if err != nil {
			publishedAt = time.Now()
		}
		
		if _, exists := existingTitles[article.Title]; exists {
			continue
		}
		entities = append(entities, &NewsEntity{
			Source:      article.Source.Name,
			Title:       article.Title,
			Description: article.Description,
			Url:         article.URL,
			UrlToImage:  article.URLToImage,
			PublishedAt: publishedAt,
		})
	}
	return
}
