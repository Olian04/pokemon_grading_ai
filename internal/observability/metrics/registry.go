package metrics

import (
	"errors"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	prometheusRegisterer prometheus.Registerer
	gatherer             prometheus.Gatherer
	httpMetrics          *HTTPMetrics
}

func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()
	mustRegisterCollector(reg, collectors.NewGoCollector())
	mustRegisterCollector(reg, collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return &Registry{
		prometheusRegisterer: reg,
		gatherer:             reg,
		httpMetrics:          NewHTTPMetrics(reg),
	}
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.gatherer, promhttp.HandlerOpts{})
}

func (r *Registry) HTTP() *HTTPMetrics {
	return r.httpMetrics
}

func mustRegisterCollector(reg prometheus.Registerer, collector prometheus.Collector) {
	if err := reg.Register(collector); err != nil {
		var alreadyRegisteredErr prometheus.AlreadyRegisteredError
		if !errors.As(err, &alreadyRegisteredErr) {
			panic(err)
		}
	}
}
