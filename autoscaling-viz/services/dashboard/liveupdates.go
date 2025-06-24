package main

import (
	_ "embed"
	"log"
	"net/http"
	"time"
)

//go:embed queries/stats.sql
var statsQuery string

type Stats struct {
	Instances      int
	Clients        int
	TotalRequests  int
	FailedRequests int
	RatePerSecond  int
	Duration       float32
}

func (s *Service) LiveUpdates(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	// Read instance count from DB and keep sending until context breaks
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			instanceCount, err := s.GetInstanceCount()
			if err != nil {
				log.Print("GetInstanceCount:", err)
				return
			}
			if err := c.WriteJSON(instanceCount); err != nil {
				log.Print("WriteJSON:", err)
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
}

// Query db for the sum of instances
func (s *Service) GetInstanceCount() (Stats, error) {
	result := Stats{}
	rows, err := s.pool.Query(statsQuery)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(
			&result.Instances,
			&result.Clients,
			&result.TotalRequests,
			&result.FailedRequests,
			&result.RatePerSecond,
			&result.Duration,
		); err != nil {
			return result, err
		}
	}
	return result, nil
}
