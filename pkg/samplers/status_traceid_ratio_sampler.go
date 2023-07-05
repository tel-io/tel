package samplers

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

var errorAttributeKey = attribute.Key("error")

func StatusTraceIDRatioBased(fraction float64) trace.Sampler {
	return statusTraceIDRatioSampler{
		traceIDRatioSampler: trace.TraceIDRatioBased(fraction),
		description:         fmt.Sprintf("StatusTraceIDRatioBased{%g}", fraction),
	}
}

type statusTraceIDRatioSampler struct {
	traceIDRatioSampler trace.Sampler
	description         string
}

func (s statusTraceIDRatioSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	res := s.traceIDRatioSampler.ShouldSample(p)

	for _, attr := range p.Attributes {
		if attr.Key != errorAttributeKey {
			continue
		}

		res.Decision = trace.RecordAndSample
		return res
	}

	for _, link := range p.Links {
		for _, attr := range link.Attributes {
			if attr.Key != errorAttributeKey {
				continue
			}

			res.Decision = trace.RecordAndSample
			return res
		}
	}

	return res
}

func (ts statusTraceIDRatioSampler) Description() string {
	return ts.description
}
