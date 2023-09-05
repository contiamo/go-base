module github.com/contiamo/go-base/v4

require (
	github.com/HdrHistogram/hdrhistogram-go v1.0.1 // indirect
	github.com/Masterminds/squirrel v1.5.4
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef // indirect
	github.com/bakins/net-http-recover v0.0.0-20141007104922-6cba69d01459
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/contiamo/jwt v0.3.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/gorilla/websocket v1.5.0
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0
	github.com/lib/pq v1.10.9
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.16.0
	github.com/robfig/cron v1.2.0
	github.com/rs/cors v1.9.0
	github.com/satori/go.uuid v1.2.0
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.8.4
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	github.com/urfave/negroni v1.0.0
	go.opentelemetry.io/contrib/detectors/aws/eks v1.6.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.36.4
	go.opentelemetry.io/contrib/instrumentation/runtime v0.31.0
	go.opentelemetry.io/contrib/propagators/aws v1.6.0
	go.opentelemetry.io/contrib/propagators/jaeger v1.6.0
	go.opentelemetry.io/otel v1.11.1
	go.opentelemetry.io/otel/exporters/jaeger v1.6.3
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.6.3
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.6.3
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.6.3
	go.opentelemetry.io/otel/exporters/prometheus v0.29.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.6.3
	go.opentelemetry.io/otel/metric v0.33.0
	go.opentelemetry.io/otel/sdk v1.6.3
	go.opentelemetry.io/otel/sdk/metric v0.29.0
	go.opentelemetry.io/otel/trace v1.11.1
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/goleak v1.2.1
	golang.org/x/crypto v0.12.0
	golang.org/x/tools v0.12.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.10.0 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	google.golang.org/genproto v0.0.0-20220407144326-9054f6ed7bac // indirect
)

go 1.14
