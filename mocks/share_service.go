package mocks

import (
	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/mock"
)

type ShareServiceMock struct {
	mock.Mock
}

func (m *ShareServiceMock) GetById(id string) (*models.Share, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Share), args.Error(1)
}

func (m *ShareServiceMock) GetByUser(userId string) ([]*models.Share, error) {
	args := m.Called(userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Share), args.Error(1)
}

func (m *ShareServiceMock) Create(share *models.Share) (*models.Share, error) {
	args := m.Called(share)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Share), args.Error(1)
}

func (m *ShareServiceMock) Delete(share *models.Share) error {
	args := m.Called(share)
	return args.Error(0)
}
