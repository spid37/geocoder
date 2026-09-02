-- Sample G-NAF rows for automated tests.
-- Source: G-NAF Core © Geoscape Australia (G-NAF EULA applies).

INSERT INTO addresses (
	address_detail_pid, address_label, number_first, street_name, street_type,
	street_name_norm, street_type_norm, locality_name, locality_name_norm,
	state, postcode, latitude, longitude, sa3_code, sa3_name
) VALUES
	('GAVIC999000001', '42 DEMO RD RICHMOND VIC 3121', '42', 'DEMO', 'RD',
	 'DEMO', 'RD', 'RICHMOND', 'RICHMOND', 'VIC', '3121',
	 -37.8182, 145.0012, '21305', 'Yarra'),
	('GAVIC999000002', '10 DEMO RD RICHMOND VIC 3121', '10', 'DEMO', 'RD',
	 'DEMO', 'RD', 'RICHMOND', 'RICHMOND', 'VIC', '3121',
	 -37.8185, 145.0015, '21305', 'Yarra'),
	('GAVIC999000003', '11 DEMO RD RICHMOND VIC 3121', '11', 'DEMO', 'RD',
	 'DEMO', 'RD', 'RICHMOND', 'RICHMOND', 'VIC', '3121',
	 -37.8184, 145.0014, '21305', 'Yarra'),
	('GAVIC424463642', '1 COLLINS ST MELBOURNE VIC 3000', '1', 'COLLINS', 'ST',
	 'COLLINS', 'ST', 'MELBOURNE', 'MELBOURNE', 'VIC', '3000',
	 -37.81363721, 144.97361666, '20604', 'Melbourne City');

INSERT INTO locality_centroids (
	state, postcode, locality_name, locality_name_norm,
	latitude, longitude, address_count, sa3_code, sa3_name
) VALUES
	('VIC', '3121', 'RICHMOND', 'RICHMOND', -37.8182, 145.0012, 12000, '21305', 'Yarra'),
	('VIC', '3000', 'MELBOURNE', 'MELBOURNE', -37.8128604016374, 144.960509621748, 119273, '20605', 'Port Phillip'),
	('VIC', '3149', 'MOUNT WAVERLEY', 'MOUNT WAVERLEY', -37.8772738823896, 145.129091190472, 19091, '21205', 'Monash'),
	('VIC', '3934', 'MOUNT MARTHA', 'MOUNT MARTHA', -38.266708994401, 145.026181848508, 10718, '21402', 'Mornington Peninsula'),
	('VIC', '3796', 'MOUNT EVELYN', 'MOUNT EVELYN', -37.783, 145.385, 4304, '21704', 'Yarra Ranges');

INSERT INTO postcode_centroids (
	state, postcode, latitude, longitude, address_count, sa3_code, sa3_name
) VALUES
	('VIC', '3121', -37.8182, 145.0012, 12000, '21305', 'Yarra'),
	('VIC', '3000', -37.8128604016374, 144.960509621748, 119273, '20605', 'Port Phillip');

INSERT INTO state_centroids (
	state, latitude, longitude, address_count, sa3_code, sa3_name
) VALUES
	('VIC', -37.7337514245503, 144.947727268739, 4184145, '21704', 'Yarra Ranges');
