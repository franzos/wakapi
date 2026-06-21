package mocks

import (
	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/mock"
)

type ShareRepositoryMock struct {
	BaseRepositoryMock
	mock.Mock
}

func (m *ShareRepositoryMock) GetById(id string) (*models.Share, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Share), args.Error(1)
}

func (m *ShareRepositoryMock) GetByUser(userId string) ([]*models.Share, error) {
	args := m.Called(userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Share), args.Error(1)
}

func (m *ShareRepositoryMock) Insert(share *models.Share) (*models.Share, error) {
	args := m.Called(share)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Share), args.Error(1)
}

func (m *ShareRepositoryMock) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
