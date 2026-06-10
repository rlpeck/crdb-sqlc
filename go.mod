module github.com/rlpeck/crdb-sqlc

go 1.26.0

toolchain go1.26.4

require (
	github.com/antlr4-go/antlr/v4 v4.13.1
	github.com/cockroachdb/cockroachdb-parser v0.25.2
	github.com/cubicdaiya/gonp v1.0.4
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/structtag v1.2.0
	github.com/go-sql-driver/mysql v1.10.0
	github.com/google/cel-go v0.28.1
	github.com/google/go-cmp v0.7.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/jinzhu/inflection v1.0.0
	github.com/lib/pq v1.12.3
	github.com/ncruces/go-sqlite3 v0.34.4
	github.com/pganalyze/pg_query_go/v6 v6.2.2
	github.com/riza-io/grpc-go v0.2.0
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/sqlc-dev/doubleclick v1.0.0
	github.com/sqlc-dev/marino v0.1.0
	github.com/tetratelabs/wazero v1.12.0
	github.com/wasilibs/go-pgquery v0.0.0-20250409022910-10ac41983c07
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/sync v0.21.0
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cel.dev/expr v0.25.1 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/bazelbuild/rules_go v0.46.0 // indirect
	github.com/biogo/store v0.0.0-20201120204734-aad293a2328f // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/apd/v3 v3.1.0 // indirect
	github.com/cockroachdb/errors v1.11.3 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/cockroachdb/version v0.0.0-20250314144055-3860cd14adf2 // indirect
	github.com/dave/dst v0.27.2 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/elastic/gosigar v0.14.3 // indirect
	github.com/getsentry/sentry-go v0.27.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/geo v0.0.0-20210211234256-740aa86cb551 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jaegertracing/jaeger v1.18.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/ncruces/go-sqlite3-wasm/v2 v2.6.35302 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/pierrre/geohash v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sasha-s/go-deadlock v0.3.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/twpayne/go-geom v1.4.1 // indirect
	github.com/twpayne/go-kml v1.5.2 // indirect
	github.com/wasilibs/wazero-helpers v0.0.0-20240620070341-3dff1577cd52 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.36.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	gonum.org/v1/gonum v0.17.0 // indirect
	google.golang.org/genproto v0.0.0-20260608224507-4308a22a1bab // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260608224507-4308a22a1bab // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260608224507-4308a22a1bab // indirect
)
