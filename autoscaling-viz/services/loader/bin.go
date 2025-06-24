package main

// func (s *Service) getCurrentLoaderConfigs() (*LoadConfig, error) {

// 	currentConfig := &LoadConfig{}

// 	// Query the database for the most recent loader config
// 	// that is currently valid.
// 	rows, err := s.pool.Query(`SELECT
// 			id, start_time, end_time, concurrency, c.k_service, href
// 			FROM loader_config c
// 			JOIN loader_services s ON c.k_service=s.k_service
// 			WHERE start_time <= CURRENT_TIMESTAMP + INTERVAL '5 seconds'
// 			AND end_time > CURRENT_TIMESTAMP
// 			ORDER BY start_time DESC LIMIT 1;`)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	if !rows.Next() {
// 		return nil, nil
// 	}
// 	if err := rows.Scan(
// 		&currentConfig.ID,
// 		&currentConfig.StartTime,
// 		&currentConfig.EndTime,
// 		&currentConfig.Concurrency,
// 		&currentConfig.ServiceKey,
// 		&currentConfig.Href); err != nil {
// 		return nil, err
// 	}

// 	// Return the slice of current loader configs.
// 	return currentConfig, nil
// }
