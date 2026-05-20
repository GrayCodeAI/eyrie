package live

// Entry is one model row from a live provider list API.
type Entry struct {
	ID                string
	DisplayName       string
	ContextWindow     int
	MaxOutput         int
	InputPricePer1M   float64
	OutputPricePer1M  float64
}
