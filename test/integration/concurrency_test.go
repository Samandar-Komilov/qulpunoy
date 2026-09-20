package integration

import (
	"sync"
	"sync/atomic"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
)

func (s *APISuite) TestConcurrentCorrectness() {
	const (
		CONCURRENCY_LOW  = 100
		CONCURRENCY_HIGH = 1000
	)

	token := s.loginSomeUser()

	s.Run("Concurrent Low 100", func() {
		pid, ok, conflict, other := s.sendConcurrentReqs(token, CONCURRENCY_LOW)

		s.Equal(int64(CONCURRENCY_LOW/2), ok)
		s.Equal(int64(CONCURRENCY_LOW/2), conflict)
		s.Equal(int64(0), other)
		s.Equal(0, s.productStock(pid))
	})

	s.Run("Concurrent High 1000", func() {
		pid, ok, conflict, other := s.sendConcurrentReqs(token, CONCURRENCY_HIGH)

		s.Equal(int64(CONCURRENCY_HIGH/2), ok)
		s.Equal(int64(CONCURRENCY_HIGH/2), conflict)
		s.Equal(int64(0), other)
		s.Equal(0, s.productStock(pid))
	})
}

func (s *APISuite) sendConcurrentReqs(token string, concurrency int) (pid uuid.UUID, ok, conflict, other int64) {
	pid = s.seedProduct(concurrency/2, "10.00")

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			resp, _ := s.createOrder(token, gofakeit.UUID(), map[string]any{
				"product_id": pid.String(),
				"quantity":   1,
			})

			switch resp.StatusCode {
			case 201:
				atomic.AddInt64(&ok, 1)
			case 409, 400:
				atomic.AddInt64(&conflict, 1)
			default:
				atomic.AddInt64(&other, 1)
			}
		}(i)
	}
	wg.Wait()

	return
}
