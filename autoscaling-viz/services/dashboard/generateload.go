package main

import (
	"fmt"
	"net/http"
	"time"
)

func (s *Service) Reset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are accepted", http.StatusMethodNotAllowed)
		return
	}

	_, err := s.pool.Exec(`UPDATE loader_instances SET reset=true`)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	time.Sleep(2 * time.Second) // Allow for propagation

	_, err = s.pool.Exec(`DELETE FROM loader_request_totals`)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
}

func (s *Service) StartTest(w http.ResponseWriter, r *http.Request) {
	// Check if POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are accepted", http.StatusMethodNotAllowed)
		return
	}

	// Grab key from path /api/start/[key]
	key := r.URL.Path[len("/api/start/"):]

	if key == "no-traffic" {
		err := s.setGlobalConcurrency(0)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "OK")
		return
	}
	if key == "regular" {
		err := s.setGlobalConcurrency(10)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "OK")
		return
	}
	if key == "surge" {
		err := s.setGlobalConcurrency(2_000)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "OK")
		return
	}
	if key == "insane" {
		err := s.setGlobalConcurrency(20_000)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "OK")
		return
	}
}

type Loader struct {
	ID          int
	Concurrency int
}

func (s *Service) retrieveLoaders() ([]Loader, error) {

	result := []Loader{}
	rows, err := s.pool.Query("SELECT id, concurrency FROM loader_instances")
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		l := Loader{}
		if err := rows.Scan(
			&l.ID,
			&l.Concurrency,
		); err != nil {
			return []Loader{}, err
		}
		result = append(result, l)
	}
	return result, nil

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func (s *Service) setGlobalConcurrency(concurrency int) error {

	loaders, err := s.retrieveLoaders()
	if err != nil {
		return err
	}

	// Fast shutdown
	if concurrency == 0 {
		for _, l := range loaders {
			err := s.setLoaderConcurrency(l.ID, 0)
			if err != nil {
				return err
			}
		}
		return nil
	}

	maxStepSize := max(10, min(concurrency/10, 1_000))
	for _, l := range loaders {
		c := min(maxStepSize, concurrency)
		concurrency = concurrency - c
		time.Sleep(time.Second * 1)
		err := s.setLoaderConcurrency(l.ID, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) setLoaderConcurrency(id int, concurrency int) error {
	_, err := s.pool.Exec(`UPDATE loader_instances 
		SET concurrency = $1 
		WHERE id = $2`, concurrency, id)
	return err
}
