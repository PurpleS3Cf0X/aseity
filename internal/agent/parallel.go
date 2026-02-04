package agent

// IsSafeToParallelize returns true if the tool is read-only and safe to run concurrently.
func IsSafeToParallelize(name string) bool {
	switch name {
	case "web_crawl", "web_search", "web_fetch", "file_read", "file_search":
		// file_read is safe if we trust OS handling, but file_search definitely is.
		// web_* are definitely safe and the primary target.
		return true
	default:
		return false
	}
}
