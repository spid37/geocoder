package store

import "database/sql"

func BuildCentroids(db *sql.DB) error {
	queries := []string{
		`INSERT INTO locality_centroids (state, postcode, locality_name, locality_name_norm, latitude, longitude, address_count)
		 SELECT state, postcode, locality_name, locality_name_norm,
		        AVG(latitude), AVG(longitude), COUNT(*)
		 FROM addresses
		 GROUP BY state, postcode, locality_name_norm`,

		`INSERT INTO postcode_centroids (state, postcode, latitude, longitude, address_count)
		 SELECT state, postcode, AVG(latitude), AVG(longitude), COUNT(*)
		 FROM addresses
		 GROUP BY state, postcode`,

		`INSERT INTO state_centroids (state, latitude, longitude, address_count)
		 SELECT state, AVG(latitude), AVG(longitude), COUNT(*)
		 FROM addresses
		 GROUP BY state`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
