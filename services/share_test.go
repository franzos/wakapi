package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
)

func setupShareTest() (*ShareService, *mocks.ShareRepositoryMock) {
	config.Set(config.Empty())
	repo := new(mocks.ShareRepositoryMock)
	return NewShareService(repo), repo
}

func TestShareService_Create(t *testing.T) {
	sut, repo := setupShareTest()
	share := &models.Share{ID: "abc", UserID: "user1", ChartType: models.ShareChartLanguages, Interval: "last_7_days", Theme: "light"}
	repo.On("Insert", share).Return(share, nil)

	result, err := sut.Create(share)

	assert.NoError(t, err)
	assert.Equal(t, "abc", result.ID)
	repo.AssertCalled(t, "Insert", share)
}

func TestShareService_GetByUser_Caches(t *testing.T) {
	sut, repo := setupShareTest()
	shares := []*models.Share{{ID: "abc", UserID: "user1"}}
	repo.On("GetByUser", "user1").Return(shares, nil).Once()

	_, _ = sut.GetByUser("user1")
	_, _ = sut.GetByUser("user1") // second call must hit cache, not repo

	repo.AssertNumberOfCalls(t, "GetByUser", 1)
}

func TestShareService_Delete_InvalidatesCache(t *testing.T) {
	sut, repo := setupShareTest()
	shares := []*models.Share{{ID: "abc", UserID: "user1"}}
	repo.On("GetByUser", "user1").Return(shares, nil)
	repo.On("Delete", "abc").Return(nil)

	_, _ = sut.GetByUser("user1") // populate cache
	err := sut.Delete(&models.Share{ID: "abc", UserID: "user1"})
	_, _ = sut.GetByUser("user1") // must re-query after invalidation

	assert.NoError(t, err)
	repo.AssertNumberOfCalls(t, "GetByUser", 2)
}
