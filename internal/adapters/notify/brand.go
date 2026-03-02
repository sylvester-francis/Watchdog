package notify

// BrandName is the brand label used in notification footers and subjects.
// Override via NOTIFICATION_BRAND environment variable.
var BrandName = "WatchDog Monitoring"

// SetBrandName overrides the default brand name used in notifications.
func SetBrandName(name string) {
	if name != "" {
		BrandName = name
	}
}
