package repository

import (
	"github.com/hasifpri/dancok"
	"gorm.io/gorm"
)

type Repository[T any] struct {
	queryGenerator *dancok.SqlGenerator
}

func NewRepositoryImpl[T any](queryGenerator *dancok.SqlGenerator) *Repository[T] {
	return &Repository[T]{
		queryGenerator: queryGenerator,
	}
}

func (r *Repository[T]) Create(db *gorm.DB, entity *T) error {
	return db.Create(entity).Error
}

func (r *Repository[T]) Update(db *gorm.DB, entity *T) error {
	return db.Save(entity).Error
}

func (r *Repository[T]) Delete(db *gorm.DB, entity *T) error {
	return db.Delete(entity).Error
}

func (r *Repository[T]) CreateBatch(db *gorm.DB, entities []*T) error {
	return db.Create(&entities).Error
}

func (r *Repository[T]) DeleteBatch(db *gorm.DB, entities []*T) error {
	return db.Delete(&entities).Error
}
