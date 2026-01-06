package engine

type HTTPRoute struct {
	Rule       Rule
	Middleware Middleware
	ServiceId  string
}
