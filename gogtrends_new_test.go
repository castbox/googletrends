package gogtrends

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDailyNew(t *testing.T) {
	// Test with invalid location
	_, err := DailyNew(context.Background(), "unknown", "Kashyyyk")
	assert.Error(t, err)

	// Test with valid parameters
	resp, err := DailyNew(context.Background(), langEN, locUS)
	assert.NoError(t, err)
	assert.True(t, len(resp) > 0)

	// Verify that the response contains valid trending searches
	for _, search := range resp {
		assert.NotNil(t, search.Title)
		assert.NotEmpty(t, search.Title.Query)
	}
}

func TestDailyTrendingSearchNew(t *testing.T) {
	// Test with invalid location
	_, err := DailyTrendingSearchNew(context.Background(), "unknown", "Kashyyyk")
	assert.Error(t, err)

	// Test with valid parameters
	resp, err := DailyTrendingSearchNew(context.Background(), langEN, locUS)
	assert.NoError(t, err)
	assert.True(t, len(resp) > 0)

	// Verify that the response contains a day with trending searches
	assert.Equal(t, "Today", resp[0].FormattedDate)
	assert.True(t, len(resp[0].Searches) > 0)

	// Verify that the trending searches have valid titles
	for _, search := range resp[0].Searches {
		assert.NotNil(t, search.Title)
		assert.NotEmpty(t, search.Title.Query)
	}
}

func TestDailyNewConcurrent(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(concurrentGoroutinesNum)

	for i := 0; i < concurrentGoroutinesNum; i++ {
		go func() {
			defer wg.Done()

			resp, err := DailyNew(context.Background(), langEN, locUS)
			assert.NoError(t, err)
			assert.True(t, len(resp) > 0)

			// Verify that the response contains valid trending searches
			for _, search := range resp {
				assert.NotNil(t, search.Title)
				assert.NotEmpty(t, search.Title.Query)
			}
		}()
	}

	wg.Wait()
}

func TestDailyTrendingSearchNewConcurrent(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(concurrentGoroutinesNum)

	for i := 0; i < concurrentGoroutinesNum; i++ {
		go func() {
			defer wg.Done()

			resp, err := DailyTrendingSearchNew(context.Background(), langEN, locUS)
			assert.NoError(t, err)
			assert.True(t, len(resp) > 0)

			// Verify that the response contains a day with trending searches
			assert.Equal(t, "Today", resp[0].FormattedDate)
			assert.True(t, len(resp[0].Searches) > 0)

			// Verify that the trending searches have valid titles
			for _, search := range resp[0].Searches {
				assert.NotNil(t, search.Title)
				assert.NotEmpty(t, search.Title.Query)
			}
		}()
	}

	wg.Wait()
}

func TestLoadDailyNew(t *testing.T) {
	res := make([][]*TrendingSearch, loadTestNum)
	errors := make([]error, loadTestNum)

	for i := 0; i < loadTestNum; i++ {
		res[i], errors[i] = DailyNew(context.Background(), langEN, locUS)
	}

	for _, e := range errors {
		assert.NoError(t, e)
	}

	for _, r := range res {
		assert.True(t, len(r) > 0)
		assert.NotEmpty(t, r[0].Title.Query)
	}
}

func TestLoadDailyTrendingSearchNew(t *testing.T) {
	res := make([][]*TrendingSearchDays, loadTestNum)
	errors := make([]error, loadTestNum)

	for i := 0; i < loadTestNum; i++ {
		res[i], errors[i] = DailyTrendingSearchNew(context.Background(), langEN, locUS)
	}

	for _, e := range errors {
		assert.NoError(t, e)
	}

	for _, r := range res {
		assert.True(t, len(r) > 0)
		assert.Equal(t, "Today", r[0].FormattedDate)
		assert.True(t, len(r[0].Searches) > 0)
		assert.NotEmpty(t, r[0].Searches[0].Title.Query)
	}
}
