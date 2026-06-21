package repositories

import (
	"errors"

	"gorm.io/gorm"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
)

type ShareRepository struct {
	BaseRepository
	config *config.Config
}

func NewShareRepository(db *gorm.DB) *ShareRepository {
	return &ShareRepository{BaseRepository: NewBaseRepository(db), config: config.Get()}
}

func (r *ShareRepository) GetById(id string) (*models.Share, error) {
	share := &models.Share{}
	if err := r.db.Preload("User").Where("id = ?", id).First(share).Error; err != nil {
		return nil, err
	}
	return share, nil
}

func (r *ShareRepository) GetByUser(userId string) ([]*models.Share, error) {
	if userId == "" {
		return []*models.Share{}, nil
	}
	var shares []*models.Share
	if err := r.db.
		Where(&models.Share{UserID: userId}).
		Order("created_at desc").
		Find(&shares).Error; err != nil {
		return shares, err
	}
	return shares, nil
}

func (r *ShareRepository) Insert(share *models.Share) (*models.Share, error) {
	if !share.IsValid() {
		return nil, errors.New("invalid share")
	}
	if err := r.db.Create(share).Error; err != nil {
		return nil, err
	}
	return share, nil
}

func (r *ShareRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(models.Share{}).Error
}
