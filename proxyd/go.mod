module github.com/aidanhopper/reverse-proxy/proxyd

go 1.25.1

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require github.com/aidanhopper/reverse-proxy/proxy-engine v0.0.0

replace github.com/aidanhopper/reverse-proxy/proxy-engine => ../proxy-engine
