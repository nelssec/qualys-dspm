# Data Classification Rules

This document defines the classification rules for detecting sensitive data.

---

## Classification Categories

| Category | Description | Regulations |
|----------|-------------|-------------|
| **PII** | Personally Identifiable Information | GDPR, CCPA, LGPD |
| **PHI** | Protected Health Information | HIPAA, HITECH |
| **PCI** | Payment Card Industry Data | PCI-DSS |
| **SECRETS** | API keys, credentials, tokens | SOC2, internal |
| **CUSTOM** | Organization-specific patterns | Custom policies |

---

## Sensitivity Levels

| Level | Description | Examples |
|-------|-------------|----------|
| **CRITICAL** | Immediate breach risk | SSN, credit cards, private keys |
| **HIGH** | Significant privacy impact | Medical records, financial data |
| **MEDIUM** | Moderate risk | Email, phone, names |
| **LOW** | Minimal risk | Public records, general metadata |

---

## Classification Rules

### PII - Personally Identifiable Information

#### SSN (Social Security Number)
```yaml
name: SSN
category: PII
sensitivity: CRITICAL
patterns:
  - '\b\d{3}-\d{2}-\d{4}\b'           # 123-45-6789
  - '\b\d{3}\s\d{2}\s\d{4}\b'         # 123 45 6789
  - '\b\d{9}\b'                        # 123456789 (with context)
validators:
  - valid_ssn_area_number              # First 3 digits validation
context_required: true                 # Needs surrounding context for bare 9-digit
```

#### Email Address
```yaml
name: EMAIL
category: PII
sensitivity: MEDIUM
patterns:
  - '\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'
validators:
  - valid_tld                          # Check for valid TLD
```

#### Phone Number
```yaml
name: PHONE
category: PII
sensitivity: MEDIUM
patterns:
  - '\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b'           # US format
  - '\b\+\d{1,3}[-.\s]?\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b'  # International
  - '\b\(\d{3}\)\s?\d{3}[-.\s]?\d{4}\b'           # (123) 456-7890
```

#### Name (requires context)
```yaml
name: PERSON_NAME
category: PII
sensitivity: MEDIUM
detection: NLP                         # Requires NLP model
context_patterns:
  - 'name:\s*'
  - 'customer:\s*'
  - 'patient:\s*'
  - 'employee:\s*'
```

#### Address
```yaml
name: ADDRESS
category: PII
sensitivity: MEDIUM
patterns:
  - '\b\d+\s+[A-Za-z]+\s+(Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Drive|Dr|Lane|Ln|Way|Court|Ct)\b'
  - '\b[A-Za-z]+,\s*[A-Z]{2}\s+\d{5}(-\d{4})?\b'  # City, ST 12345
validators:
  - valid_us_state
  - valid_zip_code
```

#### Date of Birth
```yaml
name: DOB
category: PII
sensitivity: HIGH
patterns:
  - '\b(0?[1-9]|1[0-2])[-/](0?[1-9]|[12]\d|3[01])[-/](19|20)\d{2}\b'  # MM/DD/YYYY
  - '\b(19|20)\d{2}[-/](0?[1-9]|1[0-2])[-/](0?[1-9]|[12]\d|3[01])\b'  # YYYY-MM-DD
context_patterns:
  - 'dob'
  - 'birth'
  - 'born'
```

#### Driver's License
```yaml
name: DRIVERS_LICENSE
category: PII
sensitivity: HIGH
patterns:
  # State-specific patterns (examples)
  - '\b[A-Z]\d{7}\b'                   # California
  - '\b\d{3}-\d{3}-\d{3}\b'            # Some states
context_required: true
```

#### Passport Number
```yaml
name: PASSPORT
category: PII
sensitivity: CRITICAL
patterns:
  - '\b[A-Z]{1,2}\d{6,9}\b'            # Various formats
context_patterns:
  - 'passport'
  - 'travel document'
```

---

### PHI - Protected Health Information

#### Medical Record Number
```yaml
name: MRN
category: PHI
sensitivity: CRITICAL
patterns:
  - '\bMRN[:\s#]?\d{6,10}\b'
  - '\bmedical\s*record[:\s#]?\d{6,10}\b'
case_insensitive: true
```

#### Diagnosis/ICD Codes
```yaml
name: ICD_CODE
category: PHI
sensitivity: HIGH
patterns:
  - '\b[A-TV-Z]\d{2}(\.\d{1,4})?\b'    # ICD-10 format
context_patterns:
  - 'diagnosis'
  - 'icd'
  - 'condition'
```

#### Prescription/NDC
```yaml
name: NDC
category: PHI
sensitivity: HIGH
patterns:
  - '\b\d{5}-\d{4}-\d{2}\b'            # NDC format: 12345-1234-12
  - '\b\d{11}\b'                        # NDC without dashes (with context)
context_patterns:
  - 'ndc'
  - 'drug'
  - 'medication'
  - 'prescription'
```

#### Health Insurance ID
```yaml
name: HEALTH_INSURANCE_ID
category: PHI
sensitivity: HIGH
patterns:
  - '\b[A-Z]{3}\d{9}\b'                # Common format
context_patterns:
  - 'insurance'
  - 'member id'
  - 'policy'
  - 'subscriber'
```

---

### PCI - Payment Card Industry

#### Credit Card Number
```yaml
name: CREDIT_CARD
category: PCI
sensitivity: CRITICAL
patterns:
  # Visa
  - '\b4\d{3}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b'
  # Mastercard
  - '\b5[1-5]\d{2}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b'
  - '\b2[2-7]\d{2}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b'
  # Amex
  - '\b3[47]\d{2}[-\s]?\d{6}[-\s]?\d{5}\b'
  # Discover
  - '\b6(?:011|5\d{2})[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b'
validators:
  - luhn_checksum                       # Luhn algorithm validation
```

#### CVV/CVC
```yaml
name: CVV
category: PCI
sensitivity: CRITICAL
patterns:
  - '\b\d{3,4}\b'                       # Only with context
context_patterns:
  - 'cvv'
  - 'cvc'
  - 'security code'
  - 'card verification'
context_required: true
```

#### Bank Account Number
```yaml
name: BANK_ACCOUNT
category: PCI
sensitivity: HIGH
patterns:
  - '\b\d{8,17}\b'                      # With context
context_patterns:
  - 'account number'
  - 'bank account'
  - 'checking'
  - 'savings'
context_required: true
```

#### Routing Number
```yaml
name: ROUTING_NUMBER
category: PCI
sensitivity: HIGH
patterns:
  - '\b\d{9}\b'
context_patterns:
  - 'routing'
  - 'aba'
  - 'transit'
validators:
  - valid_aba_routing                   # ABA checksum validation
```

#### IBAN
```yaml
name: IBAN
category: PCI
sensitivity: HIGH
patterns:
  - '\b[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z0-9]?){0,16}\b'
validators:
  - valid_iban_checksum
```

---

### SECRETS - Credentials and Keys

#### AWS Access Key
```yaml
name: AWS_ACCESS_KEY
category: SECRETS
sensitivity: CRITICAL
patterns:
  - '\bAKIA[0-9A-Z]{16}\b'             # Access Key ID
  - '\bASIA[0-9A-Z]{16}\b'             # Temporary Access Key
```

#### AWS Secret Key
```yaml
name: AWS_SECRET_KEY
category: SECRETS
sensitivity: CRITICAL
patterns:
  - '\b[A-Za-z0-9/+=]{40}\b'           # With context
context_patterns:
  - 'aws_secret'
  - 'secret_access_key'
  - 'secretaccesskey'
```

#### Private Key
```yaml
name: PRIVATE_KEY
category: SECRETS
sensitivity: CRITICAL
patterns:
  - '-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----'
  - '-----BEGIN PGP PRIVATE KEY BLOCK-----'
```

#### API Key (Generic)
```yaml
name: API_KEY
category: SECRETS
sensitivity: HIGH
patterns:
  - '\b[a-zA-Z0-9]{32,64}\b'           # With context
context_patterns:
  - 'api_key'
  - 'apikey'
  - 'api-key'
  - 'x-api-key'
  - 'authorization'
```

#### JWT Token
```yaml
name: JWT
category: SECRETS
sensitivity: HIGH
patterns:
  - '\beyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_.+/=]+\b'
```

#### GitHub Token
```yaml
name: GITHUB_TOKEN
category: SECRETS
sensitivity: CRITICAL
patterns:
  - '\bghp_[A-Za-z0-9]{36}\b'          # Personal access token
  - '\bgho_[A-Za-z0-9]{36}\b'          # OAuth token
  - '\bghu_[A-Za-z0-9]{36}\b'          # User-to-server token
  - '\bghs_[A-Za-z0-9]{36}\b'          # Server-to-server token
  - '\bghr_[A-Za-z0-9]{36}\b'          # Refresh token
```

#### Slack Token
```yaml
name: SLACK_TOKEN
category: SECRETS
sensitivity: HIGH
patterns:
  - '\bxox[baprs]-[0-9A-Za-z-]{10,}\b'
```

#### Google API Key
```yaml
name: GOOGLE_API_KEY
category: SECRETS
sensitivity: HIGH
patterns:
  - '\bAIza[0-9A-Za-z-_]{35}\b'
```

#### Azure Connection String
```yaml
name: AZURE_CONNECTION_STRING
category: SECRETS
sensitivity: CRITICAL
patterns:
  - 'DefaultEndpointsProtocol=https?;AccountName=[^;]+;AccountKey=[A-Za-z0-9+/=]+'
  - 'AccountKey=[A-Za-z0-9+/=]{86}=='
```

#### Database Connection String
```yaml
name: DB_CONNECTION_STRING
category: SECRETS
sensitivity: CRITICAL
patterns:
  - '\b(mysql|postgresql|mongodb|redis)://[^:]+:[^@]+@'
  - 'password=[^&;\s]+'
  - 'pwd=[^&;\s]+'
```

---

## Sampling Strategy

For large datasets, intelligent sampling prevents scanning petabytes:

```yaml
sampling:
  max_file_size: 104857600              # 100MB - skip larger files
  sample_bytes: 1048576                  # 1MB sample per file
  files_per_bucket: 1000                 # Max files per container
  random_sample_pct: 0.10                # 10% random sampling for large buckets

  priority_extensions:
    high:
      - .csv
      - .json
      - .xlsx
      - .parquet
      - .sql
      - .log
      - .txt
    medium:
      - .xml
      - .yaml
      - .yml
      - .md
      - .tsv
    skip:
      - .jpg
      - .jpeg
      - .png
      - .gif
      - .mp4
      - .mp3
      - .zip
      - .gz
      - .tar
      - .exe
      - .dll
      - .so

  sampling_regions:
    - offset: 0                          # Beginning
      size: 524288                       # 512KB
    - offset: middle                     # Middle of file
      size: 524288
    - offset: end                        # End of file (last 512KB)
      size: 524288
```

---

## Validation Functions

### Luhn Algorithm (Credit Cards)

```go
func luhnCheck(number string) bool {
    // Remove non-digits
    digits := regexp.MustCompile(`\D`).ReplaceAllString(number, "")

    sum := 0
    alternate := false

    for i := len(digits) - 1; i >= 0; i-- {
        n, _ := strconv.Atoi(string(digits[i]))

        if alternate {
            n *= 2
            if n > 9 {
                n = n%10 + 1
            }
        }

        sum += n
        alternate = !alternate
    }

    return sum%10 == 0
}
```

### SSN Area Number Validation

```go
func validSSNArea(ssn string) bool {
    // First 3 digits (area number) validation
    area, _ := strconv.Atoi(ssn[:3])

    // Invalid area numbers
    if area == 0 || area == 666 || area >= 900 {
        return false
    }

    return true
}
```

### IBAN Checksum

```go
func validIBAN(iban string) bool {
    // Move first 4 chars to end
    rearranged := iban[4:] + iban[:4]

    // Convert letters to numbers (A=10, B=11, etc.)
    numeric := ""
    for _, c := range strings.ToUpper(rearranged) {
        if c >= 'A' && c <= 'Z' {
            numeric += strconv.Itoa(int(c - 'A' + 10))
        } else {
            numeric += string(c)
        }
    }

    // Modulo 97 check
    remainder := new(big.Int)
    n, _ := new(big.Int).SetString(numeric, 10)
    remainder.Mod(n, big.NewInt(97))

    return remainder.Int64() == 1
}
```

---

## Redaction

All sample matches should be redacted before storage:

```go
func redact(value string) string {
    if len(value) <= 4 {
        return "****"
    }
    return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// Examples:
// "4532015112830366" -> "45**********66"
// "john@example.com" -> "jo**********om"
// "123-45-6789"      -> "12*******89"
```

---

## Future: ML-Enhanced Classification

For unstructured data and context-aware detection:

- Named Entity Recognition (NER) for person names
- Document classification (medical records, financial statements)
- Custom model training for organization-specific data
- Confidence scoring with human-in-the-loop review
