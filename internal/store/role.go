package store

const (
	Operator  Role = 10
	Admin     Role = 1_000
	Superuser Role = 10_000
)

type Role int64

func (r Role) ToString() string {
	switch r {
	case Superuser:
		return "superuser"
	case Admin:
		return "admin"
	default:
		return "operator"
	}
}

func ListNewUserRoles() []Role {
	return []Role{
		Operator,
		Admin,
	}
}
