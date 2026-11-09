package model

// CanResetDestination excludes any task that may have files or a resume archive.
func (j Job) CanResetDestination() bool {
	return j.Started == "" && j.Bytes == 0 && j.Files == 0 && j.Skipped == 0 && j.Status != "running" && j.Status != "completed" && j.Status != "cancelled"
}
func SiteDestination(s Settings, site string) string {
	if d := s.Sites[site].Destination; d != "" {
		return d
	}
	return s.Destination
}
func (j Job) EffectiveDestination(s Settings) string {
	if j.DestinationMode == "inherit" && j.CanResetDestination() {
		return SiteDestination(s, j.Site)
	}
	return j.Destination
}
