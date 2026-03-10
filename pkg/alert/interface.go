package alert

import "time"

type Alert struct {
	Name        string
	FiringAt    time.Time
	ResolvedAt  time.Time
	Status      string
	Severity    string
	Labels      map[string]string
	Annotations map[string]string
	Value       float64
	Fingerprint string
}

func (a *Alert) IsFiring() bool { return a.Status == "firing" }
func (a *Alert) GetLabel(key string) string {
	if a.Labels == nil { return "" }
	return a.Labels[key]
}
