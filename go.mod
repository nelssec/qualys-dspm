module github.com/qualys/dspm

go 1.22

require (
	// AWS SDK
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/credentials v1.16.12
	github.com/aws/aws-sdk-go-v2/service/iam v1.28.5
	github.com/aws/aws-sdk-go-v2/service/kms v1.27.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.49.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.5
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5

	// Azure SDK
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2 v2.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.5.0
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.2.0

	// GCP SDK
	cloud.google.com/go/iam v1.1.5
	cloud.google.com/go/storage v1.35.1
	google.golang.org/api v0.154.0

	// Database
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.9
	github.com/neo4j/neo4j-go-driver/v5 v5.15.0

	// Redis
	github.com/redis/go-redis/v9 v9.4.0

	// HTTP
	github.com/go-chi/chi/v5 v5.0.11

	// Auth
	github.com/golang-jwt/jwt/v5 v5.2.0
	golang.org/x/crypto v0.18.0

	// Scheduler
	github.com/robfig/cron/v3 v3.0.1

	// PDF Generation
	github.com/jung-kurt/gofpdf v1.16.2

	// Utils
	github.com/google/uuid v1.5.0
	gopkg.in/yaml.v3 v3.0.1
)
