module example.com/thing

go 1.25.1

require github.com/aidanhopper/reverse-proxy/proxy-engine v0.0.0
replace github.com/aidanhopper/reverse-proxy/proxy-engine => ./proxy-engine

require github.com/aidanhopper/reverse-proxy/proxyd v0.0.0
replace github.com/aidanhopper/reverse-proxy/proxyd => ./proxyd
