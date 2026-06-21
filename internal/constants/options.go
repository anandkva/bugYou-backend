package constants

const (
	RoleUser      = "user"
	RoleDeveloper = "developer"
	RoleAdmin     = "admin"

	StatusOpen     = "Open"
	StatusOnHold   = "On Hold"
	StatusResolved = "Resolved"
	StatusRejected = "Rejected"
)

var Products = []string{
	"ZenClass",
	"Classify",
	"Hyernet",
	"PlacementInfo",
	"GuviPortal",
	"Other",
}

var IssueTypes = []string{"Bug", "New Requirement"}

var Priorities = []string{"Low", "Medium", "High", "Critical"}

var Categories = []string{
	"UI",
	"Backend/API",
	"Login/Auth",
	"Payment",
	"Performance",
	"Data Issue",
	"New Requirement",
	"Other",
}

var Statuses = []string{StatusOpen, StatusOnHold, StatusResolved, StatusRejected}

func IsAllowed(value string, allowed []string) bool {
	for _, option := range allowed {
		if value == option {
			return true
		}
	}

	return false
}
