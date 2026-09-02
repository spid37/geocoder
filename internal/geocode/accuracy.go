package geocode

type Accuracy string

const (
	AccuracyStreet   Accuracy = "street"
	AccuracySuburb   Accuracy = "suburb"
	AccuracyPostcode Accuracy = "postcode"
	AccuracyState    Accuracy = "state"
)
