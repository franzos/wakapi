package services

import (
	"errors"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/repositories"
)

type ShareService struct {
	config     *config.Config
	cache      *cache.Cache
	repository repositories.IShareRepository
}

func NewShareService(shareRepository repositories.IShareRepository) *ShareService {
	return &ShareService{
		config:     config.Get(),
		repository: shareRepository,
		cache:      cache.New(24*time.Hour, 24*time.Hour),
	}
}

func (srv *ShareService) GetById(id string) (*models.Share, error) {
	cacheKey := "id_" + id
	if cached, found := srv.cache.Get(cacheKey); found {
		return cached.(*models.Share), nil
	}
	share, err := srv.repository.GetById(id)
	if err != nil {
		return nil, err
	}
	srv.cache.Set(cacheKey, share, cache.DefaultExpiration)
	return share, nil
}

func (srv *ShareService) GetByUser(userId string) ([]*models.Share, error) {
	if shares, found := srv.cache.Get(userId); found {
		return shares.([]*models.Share), nil
	}
	shares, err := srv.repository.GetByUser(userId)
	if err != nil {
		return nil, err
	}
	srv.cache.Set(userId, shares, cache.DefaultExpiration)
	return shares, nil
}

func (srv *ShareService) Create(share *models.Share) (*models.Share, error) {
	result, err := srv.repository.Insert(share)
	if err != nil {
		return nil, err
	}
	srv.cache.Delete(result.UserID)
	return result, nil
}

func (srv *ShareService) Delete(share *models.Share) error {
	if share.UserID == "" {
		return errors.New("no user id specified")
	}
	err := srv.repository.Delete(share.ID)
	srv.cache.Delete(share.UserID)
	srv.cache.Delete("id_" + share.ID)
	return err
}
