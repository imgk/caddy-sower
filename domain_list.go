package sower

// DomainList is ...
type DomainList []string

// Match is ...
func (lst *DomainList) Match(name string) bool {
	for _, v := range *lst {
		if name == v {
			return true
		}
	}
	return false
}
