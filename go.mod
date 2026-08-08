module github.com/larsartmann/go-health-dashboard

go 1.26.5

require (
	github.com/a-h/templ v0.3.1020
	github.com/larsartmann/go-datastar v0.0.3
	github.com/larsartmann/go-health v0.0.1
	github.com/larsartmann/go-sse v0.4.0
	github.com/larsartmann/templ-components v1.7.0
	github.com/samber/do/v2 v2.1.0
)

require (
	github.com/Oudwins/tailwind-merge-go v0.2.3 // indirect
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/samber/go-type-to-string v1.8.0 // indirect
)

replace github.com/larsartmann/go-health => ../go-health

replace github.com/larsartmann/templ-components => ../templ-components
