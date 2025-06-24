package trace

// 全局配置Email
var Email string

// 全局配置Email
func SetEmail(email string) {
	Email = email
}

// 全局配置Email
func GetEmail() string {
	return Email
}
