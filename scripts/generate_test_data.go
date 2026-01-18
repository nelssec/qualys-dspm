package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	firstNames = []string{"John", "Jane", "Michael", "Emily", "David", "Sarah", "Robert", "Lisa", "William", "Jennifer", "James", "Maria", "Charles", "Patricia", "Thomas", "Linda", "Daniel", "Elizabeth", "Matthew", "Barbara"}
	lastNames  = []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin"}
	streets    = []string{"Main St", "Oak Ave", "Maple Dr", "Cedar Ln", "Pine Rd", "Elm St", "Washington Blvd", "Park Ave", "Lake Dr", "River Rd"}
	cities     = []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose"}
	states     = []string{"NY", "CA", "IL", "TX", "AZ", "PA", "FL", "OH", "GA", "NC"}
	diagnoses  = []string{"Type 2 Diabetes Mellitus", "Essential Hypertension", "Major Depressive Disorder", "Generalized Anxiety Disorder", "Chronic Obstructive Pulmonary Disease", "Coronary Artery Disease", "Rheumatoid Arthritis", "Hypothyroidism", "Asthma", "Chronic Kidney Disease"}
	medications = []string{"Metformin 500mg", "Lisinopril 10mg", "Atorvastatin 20mg", "Omeprazole 20mg", "Amlodipine 5mg", "Metoprolol 25mg", "Albuterol Inhaler", "Levothyroxine 50mcg", "Gabapentin 300mg", "Sertraline 50mg"}
	departments = []string{"Engineering", "Sales", "Marketing", "Finance", "Human Resources", "Legal", "Operations", "Customer Support", "Research", "Product"}
	companies   = []string{"Acme Corp", "TechStart Inc", "Global Systems LLC", "DataFlow Partners", "CloudNine Solutions", "Innovate Labs", "SecureNet Corp", "QuantumLeap Inc", "BlueSky Ventures", "NextGen Technologies"}
)

func main() {
	rand.Seed(time.Now().UnixNano())

	bucketName := os.Getenv("DSPM_TEST_BUCKET")
	if bucketName == "" {
		bucketName = "dspm-test-data-314104994032"
	}

	outputDir := "/tmp/dspm-test-data"
	os.MkdirAll(outputDir, 0755)

	fmt.Println("Generating 500 test documents...")

	// Generate documents by category
	docs := []struct {
		category string
		count    int
		gen      func(int) (string, string, string)
	}{
		{"pii/employees", 50, generateEmployeeRecord},
		{"pii/customers", 50, generateCustomerData},
		{"pii/contacts", 30, generateContactList},
		{"phi/medical_records", 50, generateMedicalRecord},
		{"phi/prescriptions", 30, generatePrescription},
		{"phi/lab_results", 30, generateLabResults},
		{"pci/transactions", 50, generateTransactionLog},
		{"pci/invoices", 30, generateInvoice},
		{"pci/bank_statements", 20, generateBankStatement},
		{"secrets/config_files", 30, generateConfigFile},
		{"secrets/env_files", 20, generateEnvFile},
		{"mixed/reports", 40, generateMixedReport},
		{"clean/documentation", 40, generateCleanDoc},
		{"clean/logs", 30, generateCleanLog},
	}

	var allFiles []string
	docNum := 1
	for _, d := range docs {
		fmt.Printf("  Generating %d %s documents...\n", d.count, d.category)
		categoryDir := filepath.Join(outputDir, d.category)
		os.MkdirAll(categoryDir, 0755)

		for i := 0; i < d.count; i++ {
			filename, content, ext := d.gen(docNum)
			fullPath := filepath.Join(categoryDir, filename+ext)
			os.WriteFile(fullPath, []byte(content), 0644)
			allFiles = append(allFiles, fullPath)
			docNum++
		}
	}

	fmt.Printf("\nGenerated %d documents in %s\n", len(allFiles), outputDir)

	// Upload to S3
	fmt.Printf("\nUploading to s3://%s...\n", bucketName)

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("us-east-1"))
	if err != nil {
		fmt.Printf("Error loading AWS config: %v\n", err)
		os.Exit(1)
	}

	client := s3.NewFromConfig(cfg)

	uploaded := 0
	for _, file := range allFiles {
		relPath, _ := filepath.Rel(outputDir, file)
		key := "test-data/" + relPath

		content, _ := os.ReadFile(file)
		contentType := getContentType(file)

		_, err := client.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket:      &bucketName,
			Key:         &key,
			Body:        strings.NewReader(string(content)),
			ContentType: &contentType,
		})
		if err != nil {
			fmt.Printf("  Error uploading %s: %v\n", key, err)
		} else {
			uploaded++
			if uploaded%50 == 0 {
				fmt.Printf("  Uploaded %d/%d files...\n", uploaded, len(allFiles))
			}
		}
	}

	fmt.Printf("\nSuccessfully uploaded %d documents to s3://%s/test-data/\n", uploaded, bucketName)

	// Print summary
	fmt.Println("\nDocument Categories:")
	fmt.Println("  PII: 130 documents (employees, customers, contacts)")
	fmt.Println("  PHI: 110 documents (medical records, prescriptions, lab results)")
	fmt.Println("  PCI: 100 documents (transactions, invoices, bank statements)")
	fmt.Println("  Secrets: 50 documents (config files, env files)")
	fmt.Println("  Mixed: 40 documents (reports with multiple data types)")
	fmt.Println("  Clean: 70 documents (no sensitive data)")
}

func getContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".txt":
		return "text/plain"
	case ".xml":
		return "application/xml"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".env":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// --- PII Generators ---

func generateEmployeeRecord(n int) (string, string, string) {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]
	ssn := fmt.Sprintf("%03d-%02d-%04d", rand.Intn(900)+100, rand.Intn(100), rand.Intn(10000))
	email := fmt.Sprintf("%s.%s@%s", strings.ToLower(first), strings.ToLower(last), strings.ToLower(strings.ReplaceAll(companies[rand.Intn(len(companies))], " ", "")+".com"))
	phone := fmt.Sprintf("(%03d) %03d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(10000))
	dob := fmt.Sprintf("%d-%02d-%02d", 1960+rand.Intn(40), rand.Intn(12)+1, rand.Intn(28)+1)
	salary := 50000 + rand.Intn(150000)
	street := fmt.Sprintf("%d %s", rand.Intn(9999)+1, streets[rand.Intn(len(streets))])
	city := cities[rand.Intn(len(cities))]
	state := states[rand.Intn(len(states))]
	zip := fmt.Sprintf("%05d", rand.Intn(90000)+10000)

	record := map[string]interface{}{
		"employee_id":    fmt.Sprintf("EMP%06d", n),
		"first_name":     first,
		"last_name":      last,
		"ssn":            ssn,
		"date_of_birth":  dob,
		"email":          email,
		"phone":          phone,
		"department":     departments[rand.Intn(len(departments))],
		"salary":         salary,
		"hire_date":      fmt.Sprintf("20%02d-%02d-%02d", rand.Intn(24), rand.Intn(12)+1, rand.Intn(28)+1),
		"address": map[string]string{
			"street": street,
			"city":   city,
			"state":  state,
			"zip":    zip,
		},
		"emergency_contact": map[string]string{
			"name":  firstNames[rand.Intn(len(firstNames))] + " " + lastNames[rand.Intn(len(lastNames))],
			"phone": fmt.Sprintf("(%03d) %03d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(10000)),
		},
	}

	content, _ := json.MarshalIndent(record, "", "  ")
	return fmt.Sprintf("employee_%06d", n), string(content), ".json"
}

func generateCustomerData(n int) (string, string, string) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Write([]string{"customer_id", "name", "email", "phone", "ssn", "dob", "address", "city", "state", "zip"})

	for i := 0; i < 10+rand.Intn(20); i++ {
		first := firstNames[rand.Intn(len(firstNames))]
		last := lastNames[rand.Intn(len(lastNames))]
		w.Write([]string{
			fmt.Sprintf("CUST%08d", rand.Intn(100000000)),
			first + " " + last,
			fmt.Sprintf("%s.%s@email.com", strings.ToLower(first), strings.ToLower(last)),
			fmt.Sprintf("%03d-%03d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(10000)),
			fmt.Sprintf("%03d-%02d-%04d", rand.Intn(900)+100, rand.Intn(100), rand.Intn(10000)),
			fmt.Sprintf("%02d/%02d/%d", rand.Intn(12)+1, rand.Intn(28)+1, 1950+rand.Intn(50)),
			fmt.Sprintf("%d %s", rand.Intn(9999)+1, streets[rand.Intn(len(streets))]),
			cities[rand.Intn(len(cities))],
			states[rand.Intn(len(states))],
			fmt.Sprintf("%05d", rand.Intn(90000)+10000),
		})
	}
	w.Flush()

	return fmt.Sprintf("customers_%06d", n), buf.String(), ".csv"
}

func generateContactList(n int) (string, string, string) {
	var contacts []map[string]string
	for i := 0; i < 20+rand.Intn(30); i++ {
		first := firstNames[rand.Intn(len(firstNames))]
		last := lastNames[rand.Intn(len(lastNames))]
		contacts = append(contacts, map[string]string{
			"name":    first + " " + last,
			"email":   fmt.Sprintf("%s.%s@%s.com", strings.ToLower(first), strings.ToLower(last), []string{"gmail", "yahoo", "outlook", "company"}[rand.Intn(4)]),
			"phone":   fmt.Sprintf("+1-%03d-%03d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(10000)),
			"company": companies[rand.Intn(len(companies))],
		})
	}
	content, _ := json.MarshalIndent(map[string]interface{}{"contacts": contacts}, "", "  ")
	return fmt.Sprintf("contacts_%06d", n), string(content), ".json"
}

// --- PHI Generators ---

func generateMedicalRecord(n int) (string, string, string) {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]
	mrn := fmt.Sprintf("MRN%09d", rand.Intn(1000000000))
	dob := fmt.Sprintf("%d-%02d-%02d", 1940+rand.Intn(60), rand.Intn(12)+1, rand.Intn(28)+1)

	record := fmt.Sprintf(`MEDICAL RECORD
================================================================================
Patient Name: %s %s
Medical Record Number: %s
Date of Birth: %s
Social Security Number: %03d-%02d-%04d
Insurance ID: INS%012d

DIAGNOSIS:
- Primary: %s (ICD-10: %s)
- Secondary: %s (ICD-10: %s)

CURRENT MEDICATIONS:
1. %s - Take as directed
2. %s - Take as directed
3. %s - Take as directed

VITAL SIGNS (Last Visit):
- Blood Pressure: %d/%d mmHg
- Heart Rate: %d bpm
- Temperature: %.1f°F
- Weight: %d lbs

PHYSICIAN NOTES:
Patient presents with %s. Recommended %s. Follow-up in %d weeks.

Attending Physician: Dr. %s %s
Date: %s
================================================================================
`,
		first, last, mrn, dob,
		rand.Intn(900)+100, rand.Intn(100), rand.Intn(10000),
		rand.Intn(1000000000000),
		diagnoses[rand.Intn(len(diagnoses))], fmt.Sprintf("%c%02d.%d", 'A'+rand.Intn(20), rand.Intn(100), rand.Intn(10)),
		diagnoses[rand.Intn(len(diagnoses))], fmt.Sprintf("%c%02d.%d", 'A'+rand.Intn(20), rand.Intn(100), rand.Intn(10)),
		medications[rand.Intn(len(medications))],
		medications[rand.Intn(len(medications))],
		medications[rand.Intn(len(medications))],
		110+rand.Intn(50), 60+rand.Intn(40),
		60+rand.Intn(40),
		97.0+rand.Float64()*3,
		120+rand.Intn(180),
		[]string{"ongoing symptoms", "improved condition", "stable vitals", "mild discomfort"}[rand.Intn(4)],
		[]string{"continued monitoring", "medication adjustment", "specialist referral", "lifestyle changes"}[rand.Intn(4)],
		2+rand.Intn(10),
		firstNames[rand.Intn(len(firstNames))], lastNames[rand.Intn(len(lastNames))],
		time.Now().AddDate(0, 0, -rand.Intn(365)).Format("2006-01-02"),
	)

	return fmt.Sprintf("medical_record_%06d", n), record, ".txt"
}

func generatePrescription(n int) (string, string, string) {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]

	rx := map[string]interface{}{
		"prescription_id": fmt.Sprintf("RX%010d", rand.Intn(10000000000)),
		"patient": map[string]string{
			"name":      first + " " + last,
			"dob":       fmt.Sprintf("%d-%02d-%02d", 1940+rand.Intn(60), rand.Intn(12)+1, rand.Intn(28)+1),
			"mrn":       fmt.Sprintf("MRN%09d", rand.Intn(1000000000)),
			"phone":     fmt.Sprintf("(%03d) %03d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(10000)),
		},
		"medication":   medications[rand.Intn(len(medications))],
		"ndc":          fmt.Sprintf("%05d-%04d-%02d", rand.Intn(100000), rand.Intn(10000), rand.Intn(100)),
		"quantity":     30 + rand.Intn(60),
		"refills":      rand.Intn(6),
		"instructions": "Take as directed by physician",
		"prescriber": map[string]string{
			"name":    "Dr. " + firstNames[rand.Intn(len(firstNames))] + " " + lastNames[rand.Intn(len(lastNames))],
			"npi":     fmt.Sprintf("%010d", rand.Intn(10000000000)),
			"dea":     fmt.Sprintf("A%c%07d", 'A'+rand.Intn(26), rand.Intn(10000000)),
		},
		"date_written": time.Now().AddDate(0, 0, -rand.Intn(30)).Format("2006-01-02"),
	}

	content, _ := json.MarshalIndent(rx, "", "  ")
	return fmt.Sprintf("prescription_%06d", n), string(content), ".json"
}

func generateLabResults(n int) (string, string, string) {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Write([]string{"test_id", "patient_name", "patient_dob", "mrn", "test_name", "result", "unit", "reference_range", "flag", "collected_date"})

	mrn := fmt.Sprintf("MRN%09d", rand.Intn(1000000000))
	dob := fmt.Sprintf("%d-%02d-%02d", 1940+rand.Intn(60), rand.Intn(12)+1, rand.Intn(28)+1)
	patientName := first + " " + last

	tests := []struct {
		name, unit, refRange string
		low, high            float64
	}{
		{"Glucose", "mg/dL", "70-100", 65, 250},
		{"Hemoglobin A1c", "%", "4.0-5.6", 4.0, 12.0},
		{"Total Cholesterol", "mg/dL", "<200", 120, 300},
		{"HDL Cholesterol", "mg/dL", ">40", 25, 90},
		{"LDL Cholesterol", "mg/dL", "<100", 50, 200},
		{"Triglycerides", "mg/dL", "<150", 50, 400},
		{"Creatinine", "mg/dL", "0.7-1.3", 0.5, 3.0},
		{"BUN", "mg/dL", "7-20", 5, 50},
		{"TSH", "mIU/L", "0.4-4.0", 0.1, 10.0},
		{"White Blood Cell", "K/uL", "4.5-11.0", 2.0, 20.0},
	}

	collectedDate := time.Now().AddDate(0, 0, -rand.Intn(30)).Format("2006-01-02")
	for _, t := range tests {
		result := t.low + rand.Float64()*(t.high-t.low)
		flag := ""
		if result < t.low+(t.high-t.low)*0.2 {
			flag = "L"
		} else if result > t.low+(t.high-t.low)*0.8 {
			flag = "H"
		}
		w.Write([]string{
			fmt.Sprintf("LAB%012d", rand.Intn(1000000000000)),
			patientName,
			dob,
			mrn,
			t.name,
			fmt.Sprintf("%.1f", result),
			t.unit,
			t.refRange,
			flag,
			collectedDate,
		})
	}
	w.Flush()

	return fmt.Sprintf("lab_results_%06d", n), buf.String(), ".csv"
}

// --- PCI Generators ---

func generateTransactionLog(n int) (string, string, string) {
	var transactions []map[string]interface{}

	for i := 0; i < 20+rand.Intn(30); i++ {
		cardNum := fmt.Sprintf("4%03d%04d%04d%04d", rand.Intn(1000), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000))
		transactions = append(transactions, map[string]interface{}{
			"transaction_id": fmt.Sprintf("TXN%015d", rand.Intn(1000000000000000)),
			"timestamp":      time.Now().Add(-time.Duration(rand.Intn(720)) * time.Hour).Format(time.RFC3339),
			"card_number":    cardNum,
			"card_type":      []string{"Visa", "Mastercard", "Amex", "Discover"}[rand.Intn(4)],
			"cvv":            fmt.Sprintf("%03d", rand.Intn(1000)),
			"expiry":         fmt.Sprintf("%02d/%02d", rand.Intn(12)+1, 24+rand.Intn(6)),
			"amount":         fmt.Sprintf("%.2f", 10+rand.Float64()*990),
			"currency":       "USD",
			"merchant":       companies[rand.Intn(len(companies))],
			"status":         []string{"approved", "approved", "approved", "declined"}[rand.Intn(4)],
			"cardholder":     firstNames[rand.Intn(len(firstNames))] + " " + lastNames[rand.Intn(len(lastNames))],
		})
	}

	content, _ := json.MarshalIndent(map[string]interface{}{"transactions": transactions}, "", "  ")
	return fmt.Sprintf("transactions_%06d", n), string(content), ".json"
}

func generateInvoice(n int) (string, string, string) {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]
	cardNum := fmt.Sprintf("5%03d-%04d-%04d-%04d", 100+rand.Intn(500), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000))
	routingNum := fmt.Sprintf("%09d", 100000000+rand.Intn(900000000))
	accountNum := fmt.Sprintf("%012d", rand.Intn(1000000000000))

	invoice := fmt.Sprintf(`INVOICE
================================================================================
Invoice #: INV-%06d
Date: %s
Due Date: %s

BILL TO:
%s %s
%d %s
%s, %s %05d

PAYMENT INFORMATION:
Credit Card: %s
Expiry: %02d/%02d
CVV: %03d

OR

Bank Transfer:
Routing Number: %s
Account Number: %s

ITEMS:
--------------------------------------------------------------------------------
Description                              Qty    Unit Price    Total
--------------------------------------------------------------------------------
Professional Services                    %d     $%.2f         $%.2f
Software License                         %d     $%.2f         $%.2f
Support & Maintenance                    1      $%.2f         $%.2f
--------------------------------------------------------------------------------
                                         Subtotal:           $%.2f
                                         Tax (8.5%%):          $%.2f
                                         TOTAL:              $%.2f
================================================================================
`,
		n,
		time.Now().AddDate(0, 0, -rand.Intn(30)).Format("2006-01-02"),
		time.Now().AddDate(0, 0, 30-rand.Intn(30)).Format("2006-01-02"),
		first, last,
		rand.Intn(9999)+1, streets[rand.Intn(len(streets))],
		cities[rand.Intn(len(cities))], states[rand.Intn(len(states))], rand.Intn(90000)+10000,
		cardNum,
		rand.Intn(12)+1, 24+rand.Intn(6),
		rand.Intn(1000),
		routingNum, accountNum,
		rand.Intn(10)+1, 100+rand.Float64()*400, float64(rand.Intn(10)+1)*(100+rand.Float64()*400),
		rand.Intn(5)+1, 500+rand.Float64()*1000, float64(rand.Intn(5)+1)*(500+rand.Float64()*1000),
		200+rand.Float64()*300, 200+rand.Float64()*300,
		5000+rand.Float64()*15000,
		(5000+rand.Float64()*15000)*0.085,
		(5000+rand.Float64()*15000)*1.085,
	)

	return fmt.Sprintf("invoice_%06d", n), invoice, ".txt"
}

func generateBankStatement(n int) (string, string, string) {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Write([]string{"date", "description", "amount", "balance", "account_number", "routing_number", "account_holder"})

	accountNum := fmt.Sprintf("%012d", rand.Intn(1000000000000))
	routingNum := fmt.Sprintf("%09d", 100000000+rand.Intn(900000000))
	balance := 5000 + rand.Float64()*45000

	for i := 0; i < 30; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		desc := []string{"Direct Deposit - Payroll", "ATM Withdrawal", "Online Transfer", "Debit Card Purchase", "Bill Pay - Utilities", "Check #" + fmt.Sprintf("%04d", rand.Intn(10000)), "Interest Credit"}[rand.Intn(7)]
		amount := -500 + rand.Float64()*3000
		balance += amount
		w.Write([]string{
			date,
			desc,
			fmt.Sprintf("%.2f", amount),
			fmt.Sprintf("%.2f", balance),
			accountNum,
			routingNum,
			first + " " + last,
		})
	}
	w.Flush()

	return fmt.Sprintf("bank_statement_%06d", n), buf.String(), ".csv"
}

// --- Secrets Generators ---

func generateConfigFile(n int) (string, string, string) {
	awsKey := fmt.Sprintf("AKIA%s", randomString(16))
	awsSecret := randomString(40)
	dbPassword := randomString(24)
	apiKey := randomString(32)
	jwtSecret := randomString(64)

	config := fmt.Sprintf(`# Application Configuration
# WARNING: Contains sensitive credentials

[database]
host = db.internal.company.com
port = 5432
name = production_db
user = app_service
password = %s

[aws]
access_key_id = %s
secret_access_key = %s
region = us-east-1

[api]
api_key = %s
api_secret = %s
endpoint = https://api.company.com/v1

[security]
jwt_secret = %s
encryption_key = %s

[redis]
host = redis.internal.company.com
port = 6379
password = %s

[slack]
webhook_url = https://hooks.slack.com/services/T%s/B%s/%s
bot_token = xoxb-%s-%s-%s
`,
		dbPassword,
		awsKey, awsSecret,
		apiKey, randomString(32),
		jwtSecret, randomString(32),
		randomString(16),
		randomString(9), randomString(9), randomString(24),
		randomString(12), randomString(12), randomString(24),
	)

	return fmt.Sprintf("config_%06d", n), config, ".yaml"
}

func generateEnvFile(n int) (string, string, string) {
	content := fmt.Sprintf(`# Environment Variables - DO NOT COMMIT
DATABASE_URL=postgresql://admin:%s@db.production.internal:5432/app_db
REDIS_URL=redis://:%s@redis.production.internal:6379/0
AWS_ACCESS_KEY_ID=AKIA%s
AWS_SECRET_ACCESS_KEY=%s
STRIPE_SECRET_KEY=sk_live_%s
STRIPE_PUBLISHABLE_KEY=pk_live_%s
SENDGRID_API_KEY=SG.%s.%s
GITHUB_TOKEN=ghp_%s
JWT_SECRET=%s
ENCRYPTION_KEY=%s
GOOGLE_API_KEY=AIza%s
TWILIO_AUTH_TOKEN=%s
SLACK_BOT_TOKEN=xoxb-%s-%s-%s
MONGODB_URI=mongodb+srv://admin:%s@cluster0.mongodb.net/production
`,
		randomString(20),
		randomString(16),
		randomString(16),
		randomString(40),
		randomString(24),
		randomString(24),
		randomString(22), randomString(43),
		randomString(36),
		randomString(64),
		randomString(32),
		randomString(35),
		randomString(32),
		randomString(12), randomString(12), randomString(24),
		randomString(20),
	)

	return fmt.Sprintf("env_%06d", n), content, ".env"
}

// --- Mixed Generators ---

func generateMixedReport(n int) (string, string, string) {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]

	report := fmt.Sprintf(`QUARTERLY BUSINESS REPORT
================================================================================
Report ID: RPT-%06d
Generated: %s
Classification: CONFIDENTIAL

EXECUTIVE SUMMARY
--------------------------------------------------------------------------------
This report contains sensitive employee, customer, and financial information.

EMPLOYEE METRICS
--------------------------------------------------------------------------------
Employee: %s %s
SSN: %03d-%02d-%04d
Department: %s
Performance Rating: %.1f/5.0

TOP CUSTOMER ACCOUNTS
--------------------------------------------------------------------------------
Customer: %s %s
Account #: %012d
Credit Card on File: %s
Total Revenue: $%,.2f

FINANCIAL SUMMARY
--------------------------------------------------------------------------------
Bank Account: %09d (Routing: %09d)
Q%d Revenue: $%,.2f
Q%d Expenses: $%,.2f
Net Profit: $%,.2f

API INTEGRATION STATUS
--------------------------------------------------------------------------------
AWS Access Key: AKIA%s
API Key: %s
Status: Active

SYSTEM CONFIGURATION
--------------------------------------------------------------------------------
Database Password: %s
JWT Secret: %s

================================================================================
This document contains PII, PCI, and credential information.
Handle according to data classification policies.
`,
		n,
		time.Now().Format("2006-01-02 15:04:05"),
		first, last,
		rand.Intn(900)+100, rand.Intn(100), rand.Intn(10000),
		departments[rand.Intn(len(departments))],
		3+rand.Float64()*2,
		firstNames[rand.Intn(len(firstNames))], lastNames[rand.Intn(len(lastNames))],
		rand.Intn(1000000000000),
		fmt.Sprintf("4%03d-%04d-%04d-%04d", rand.Intn(1000), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000)),
		100000+rand.Float64()*900000,
		rand.Intn(1000000000), 100000000+rand.Intn(900000000),
		rand.Intn(4)+1, 1000000+rand.Float64()*4000000,
		rand.Intn(4)+1, 500000+rand.Float64()*2000000,
		500000+rand.Float64()*2000000,
		randomString(16),
		randomString(32),
		randomString(20),
		randomString(48),
	)

	return fmt.Sprintf("quarterly_report_%06d", n), report, ".txt"
}

// --- Clean Generators ---

func generateCleanDoc(n int) (string, string, string) {
	topics := []string{
		"API Documentation",
		"Architecture Overview",
		"Deployment Guide",
		"User Manual",
		"Release Notes",
		"Code Style Guide",
		"Testing Strategy",
		"Security Best Practices",
	}

	doc := fmt.Sprintf(`# %s

## Overview

This document provides comprehensive information about our system architecture
and implementation details.

## Table of Contents

1. Introduction
2. Getting Started
3. Configuration
4. API Reference
5. Troubleshooting

## Introduction

Our platform provides a scalable solution for enterprise data management.
The system is designed with security and performance in mind.

## Architecture

The system consists of the following components:

- Frontend: React-based single page application
- Backend: Go microservices with REST APIs
- Database: PostgreSQL for relational data
- Cache: Redis for session management
- Queue: RabbitMQ for async processing

## Configuration

Configuration is managed through YAML files and environment variables.
See the configuration reference for available options.

## API Reference

### Endpoints

- GET /api/v1/resources - List all resources
- POST /api/v1/resources - Create a new resource
- GET /api/v1/resources/{id} - Get resource by ID
- PUT /api/v1/resources/{id} - Update resource
- DELETE /api/v1/resources/{id} - Delete resource

### Response Codes

- 200: Success
- 201: Created
- 400: Bad Request
- 401: Unauthorized
- 404: Not Found
- 500: Internal Server Error

## Troubleshooting

Common issues and their solutions:

1. Connection timeout: Check network configuration
2. Authentication failure: Verify credentials
3. Performance issues: Check resource utilization

## Support

For additional support, contact the engineering team.

---
Document Version: 1.%d.0
Last Updated: %s
`,
		topics[rand.Intn(len(topics))],
		rand.Intn(10),
		time.Now().Format("2006-01-02"),
	)

	return fmt.Sprintf("documentation_%06d", n), doc, ".txt"
}

func generateCleanLog(n int) (string, string, string) {
	var logs []string

	levels := []string{"INFO", "DEBUG", "WARN", "INFO", "INFO", "DEBUG"}
	messages := []string{
		"Application started successfully",
		"Processing request",
		"Cache hit for key",
		"Database query executed",
		"HTTP request completed",
		"Background job scheduled",
		"Metrics collected",
		"Health check passed",
		"Connection pool initialized",
		"Configuration loaded",
	}

	baseTime := time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour)
	for i := 0; i < 100+rand.Intn(100); i++ {
		logTime := baseTime.Add(time.Duration(i*rand.Intn(60)) * time.Second)
		level := levels[rand.Intn(len(levels))]
		msg := messages[rand.Intn(len(messages))]
		reqID := fmt.Sprintf("req-%s", randomString(8))
		logs = append(logs, fmt.Sprintf("%s [%s] [%s] %s duration=%dms",
			logTime.Format("2006-01-02T15:04:05.000Z"),
			level,
			reqID,
			msg,
			rand.Intn(500),
		))
	}

	return fmt.Sprintf("application_%06d", n), strings.Join(logs, "\n"), ".log"
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
