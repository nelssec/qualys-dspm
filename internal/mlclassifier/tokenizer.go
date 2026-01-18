package mlclassifier

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Tokenizer provides BPE tokenization for text preprocessing
type Tokenizer struct {
	vocab      map[string]int32
	merges     []BPEMerge
	specialTokens map[string]int32
	maxLength  int

	// Special token IDs
	padToken   int32
	unkToken   int32
	clsToken   int32
	sepToken   int32
	maskToken  int32

	// Pre-tokenization regex
	preTokenizeRe *regexp.Regexp
}

// BPEMerge represents a BPE merge operation
type BPEMerge struct {
	A string
	B string
}

// TokenizerConfig contains tokenizer configuration
type TokenizerConfig struct {
	VocabFile      string `json:"vocab_file"`
	MergesFile     string `json:"merges_file"`
	MaxLength      int    `json:"max_length"`
	PadToken       string `json:"pad_token"`
	UnkToken       string `json:"unk_token"`
	ClsToken       string `json:"cls_token"`
	SepToken       string `json:"sep_token"`
	MaskToken      string `json:"mask_token"`
}

// NewTokenizer creates a new BPE tokenizer
func NewTokenizer(config TokenizerConfig) (*Tokenizer, error) {
	t := &Tokenizer{
		vocab:         make(map[string]int32),
		specialTokens: make(map[string]int32),
		maxLength:     config.MaxLength,
	}

	if config.MaxLength == 0 {
		t.maxLength = 512
	}

	// Load vocabulary if provided
	if config.VocabFile != "" {
		if err := t.loadVocab(config.VocabFile); err != nil {
			return nil, err
		}
	} else {
		// Initialize with basic vocabulary
		t.initializeBasicVocab()
	}

	// Load merges if provided
	if config.MergesFile != "" {
		if err := t.loadMerges(config.MergesFile); err != nil {
			return nil, err
		}
	}

	// Set special token IDs
	t.setupSpecialTokens(config)

	// Pre-tokenization pattern (splits on whitespace and punctuation)
	t.preTokenizeRe = regexp.MustCompile(`'s|'t|'re|'ve|'m|'ll|'d| ?\p{L}+| ?\p{N}+| ?[^\s\p{L}\p{N}]+|\s+`)

	return t, nil
}

// initializeBasicVocab creates a basic vocabulary for when no vocab file is provided
func (t *Tokenizer) initializeBasicVocab() {
	// Add special tokens
	t.vocab["[PAD]"] = 0
	t.vocab["[UNK]"] = 1
	t.vocab["[CLS]"] = 2
	t.vocab["[SEP]"] = 3
	t.vocab["[MASK]"] = 4

	// Add basic ASCII characters
	idx := int32(5)
	for c := 'a'; c <= 'z'; c++ {
		t.vocab[string(c)] = idx
		idx++
	}
	for c := 'A'; c <= 'Z'; c++ {
		t.vocab[string(c)] = idx
		idx++
	}
	for c := '0'; c <= '9'; c++ {
		t.vocab[string(c)] = idx
		idx++
	}

	// Add common punctuation
	for _, c := range ".,!?;:\"'()-_@#$%&*+=/\\[]{}|<>~`" {
		t.vocab[string(c)] = idx
		idx++
	}

	// Add space
	t.vocab[" "] = idx
	idx++

	// Add common subwords
	commonSubwords := []string{
		"##s", "##ing", "##ed", "##er", "##ly", "##tion", "##able",
		"the", "and", "is", "in", "to", "of", "for", "on", "with",
		"name", "email", "phone", "address", "date", "ssn", "card",
		"number", "credit", "account", "password", "user", "id",
	}
	for _, w := range commonSubwords {
		t.vocab[w] = idx
		idx++
	}
}

// loadVocab loads vocabulary from a JSON file
func (t *Tokenizer) loadVocab(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &t.vocab)
}

// loadMerges loads BPE merges from a file
func (t *Tokenizer) loadMerges(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 2 {
			t.merges = append(t.merges, BPEMerge{A: parts[0], B: parts[1]})
		}
	}

	return nil
}

// setupSpecialTokens configures special token IDs
func (t *Tokenizer) setupSpecialTokens(config TokenizerConfig) {
	// Default special tokens
	padToken := "[PAD]"
	unkToken := "[UNK]"
	clsToken := "[CLS]"
	sepToken := "[SEP]"
	maskToken := "[MASK]"

	if config.PadToken != "" {
		padToken = config.PadToken
	}
	if config.UnkToken != "" {
		unkToken = config.UnkToken
	}
	if config.ClsToken != "" {
		clsToken = config.ClsToken
	}
	if config.SepToken != "" {
		sepToken = config.SepToken
	}
	if config.MaskToken != "" {
		maskToken = config.MaskToken
	}

	// Get or assign IDs
	t.padToken = t.getOrAssignID(padToken, 0)
	t.unkToken = t.getOrAssignID(unkToken, 1)
	t.clsToken = t.getOrAssignID(clsToken, 2)
	t.sepToken = t.getOrAssignID(sepToken, 3)
	t.maskToken = t.getOrAssignID(maskToken, 4)

	t.specialTokens[padToken] = t.padToken
	t.specialTokens[unkToken] = t.unkToken
	t.specialTokens[clsToken] = t.clsToken
	t.specialTokens[sepToken] = t.sepToken
	t.specialTokens[maskToken] = t.maskToken
}

func (t *Tokenizer) getOrAssignID(token string, defaultID int32) int32 {
	if id, ok := t.vocab[token]; ok {
		return id
	}
	return defaultID
}

// Encode tokenizes text and returns input IDs and attention mask
func (t *Tokenizer) Encode(text string) ([]int32, []int32) {
	// Lowercase and clean text
	text = strings.ToLower(strings.TrimSpace(text))

	// Pre-tokenize
	tokens := t.preTokenize(text)

	// Apply BPE
	bpeTokens := t.applyBPE(tokens)

	// Convert to IDs
	inputIDs := make([]int32, 0, len(bpeTokens)+2)

	// Add [CLS] token
	inputIDs = append(inputIDs, t.clsToken)

	// Add token IDs
	for _, token := range bpeTokens {
		if id, ok := t.vocab[token]; ok {
			inputIDs = append(inputIDs, id)
		} else {
			inputIDs = append(inputIDs, t.unkToken)
		}
	}

	// Add [SEP] token
	inputIDs = append(inputIDs, t.sepToken)

	// Create attention mask
	attentionMask := make([]int32, len(inputIDs))
	for i := range attentionMask {
		attentionMask[i] = 1
	}

	// Truncate or pad to max length
	inputIDs, attentionMask = t.padOrTruncate(inputIDs, attentionMask)

	return inputIDs, attentionMask
}

// EncodeBatch tokenizes multiple texts
func (t *Tokenizer) EncodeBatch(texts []string) ([][]int32, [][]int32) {
	inputIDs := make([][]int32, len(texts))
	attentionMasks := make([][]int32, len(texts))

	for i, text := range texts {
		inputIDs[i], attentionMasks[i] = t.Encode(text)
	}

	return inputIDs, attentionMasks
}

// preTokenize splits text into tokens
func (t *Tokenizer) preTokenize(text string) []string {
	matches := t.preTokenizeRe.FindAllString(text, -1)
	return matches
}

// applyBPE applies BPE merges to tokens
func (t *Tokenizer) applyBPE(tokens []string) []string {
	var result []string

	for _, token := range tokens {
		if token == "" {
			continue
		}

		// Split token into characters
		chars := splitIntoChars(token)

		// Apply merges
		for _, merge := range t.merges {
			chars = applyMerge(chars, merge.A, merge.B)
		}

		result = append(result, chars...)
	}

	return result
}

// splitIntoChars splits a string into individual characters/runes
func splitIntoChars(s string) []string {
	var result []string
	for i, r := range s {
		if i == 0 {
			result = append(result, string(r))
		} else {
			result = append(result, "##"+string(r))
		}
	}
	return result
}

// applyMerge applies a single BPE merge operation
func applyMerge(chars []string, a, b string) []string {
	var result []string
	i := 0
	for i < len(chars) {
		if i < len(chars)-1 && chars[i] == a && chars[i+1] == b {
			result = append(result, a+b)
			i += 2
		} else {
			result = append(result, chars[i])
			i++
		}
	}
	return result
}

// padOrTruncate adjusts sequence to max length
func (t *Tokenizer) padOrTruncate(inputIDs, attentionMask []int32) ([]int32, []int32) {
	if len(inputIDs) > t.maxLength {
		// Truncate
		inputIDs = inputIDs[:t.maxLength]
		attentionMask = attentionMask[:t.maxLength]
	} else if len(inputIDs) < t.maxLength {
		// Pad
		padding := t.maxLength - len(inputIDs)
		for i := 0; i < padding; i++ {
			inputIDs = append(inputIDs, t.padToken)
			attentionMask = append(attentionMask, 0)
		}
	}

	return inputIDs, attentionMask
}

// Decode converts token IDs back to text
func (t *Tokenizer) Decode(ids []int32) string {
	// Build reverse vocabulary
	reverseVocab := make(map[int32]string)
	for token, id := range t.vocab {
		reverseVocab[id] = token
	}

	var tokens []string
	for _, id := range ids {
		if id == t.padToken || id == t.clsToken || id == t.sepToken {
			continue
		}
		if token, ok := reverseVocab[id]; ok {
			tokens = append(tokens, token)
		}
	}

	// Join tokens and clean up
	text := strings.Join(tokens, "")
	text = strings.ReplaceAll(text, "##", "")
	text = strings.TrimSpace(text)

	return text
}

// VocabSize returns the vocabulary size
func (t *Tokenizer) VocabSize() int {
	return len(t.vocab)
}

// GetSpecialTokenIDs returns the special token IDs
func (t *Tokenizer) GetSpecialTokenIDs() map[string]int32 {
	return map[string]int32{
		"pad":  t.padToken,
		"unk":  t.unkToken,
		"cls":  t.clsToken,
		"sep":  t.sepToken,
		"mask": t.maskToken,
	}
}

// SimpleTokenizer provides a simpler tokenization approach for when ONNX models are not available
type SimpleTokenizer struct {
	vocab     map[string]int32
	maxLength int
}

// NewSimpleTokenizer creates a word-based tokenizer
func NewSimpleTokenizer(maxLength int) *SimpleTokenizer {
	if maxLength == 0 {
		maxLength = 512
	}

	t := &SimpleTokenizer{
		vocab:     make(map[string]int32),
		maxLength: maxLength,
	}

	// Initialize vocabulary with common tokens
	t.initVocab()

	return t
}

func (t *SimpleTokenizer) initVocab() {
	// Special tokens
	t.vocab["[PAD]"] = 0
	t.vocab["[UNK]"] = 1
	t.vocab["[CLS]"] = 2
	t.vocab["[SEP]"] = 3

	// Common words for PII detection
	words := []string{
		"name", "email", "phone", "address", "date", "birth", "ssn",
		"social", "security", "number", "credit", "card", "account",
		"password", "user", "username", "id", "identifier", "license",
		"driver", "passport", "bank", "routing", "medical", "health",
		"diagnosis", "patient", "doctor", "hospital", "insurance",
		"salary", "income", "tax", "financial", "confidential", "private",
	}

	idx := int32(4)
	for _, word := range words {
		t.vocab[word] = idx
		idx++
	}
}

// Tokenize splits text into tokens
func (t *SimpleTokenizer) Tokenize(text string) []string {
	// Normalize
	text = strings.ToLower(text)

	// Split on whitespace and punctuation
	re := regexp.MustCompile(`[\s\p{P}]+`)
	tokens := re.Split(text, -1)

	// Filter empty tokens
	var result []string
	for _, token := range tokens {
		if token != "" {
			result = append(result, token)
		}
	}

	return result
}

// Encode converts text to token IDs
func (t *SimpleTokenizer) Encode(text string) ([]int32, []int32) {
	tokens := t.Tokenize(text)

	inputIDs := make([]int32, 0, len(tokens)+2)
	inputIDs = append(inputIDs, 2) // [CLS]

	for _, token := range tokens {
		if id, ok := t.vocab[token]; ok {
			inputIDs = append(inputIDs, id)
		} else {
			inputIDs = append(inputIDs, 1) // [UNK]
		}
	}

	inputIDs = append(inputIDs, 3) // [SEP]

	// Pad or truncate
	attentionMask := make([]int32, len(inputIDs))
	for i := range attentionMask {
		attentionMask[i] = 1
	}

	for len(inputIDs) < t.maxLength {
		inputIDs = append(inputIDs, 0)
		attentionMask = append(attentionMask, 0)
	}

	if len(inputIDs) > t.maxLength {
		inputIDs = inputIDs[:t.maxLength]
		attentionMask = attentionMask[:t.maxLength]
	}

	return inputIDs, attentionMask
}

// GetWordImportance returns token importance scores using TF-IDF style scoring
func GetWordImportance(text string, targetWords []string) map[string]float64 {
	importance := make(map[string]float64)
	text = strings.ToLower(text)
	words := strings.Fields(text)
	totalWords := float64(len(words))

	if totalWords == 0 {
		return importance
	}

	// Count word frequencies
	wordFreq := make(map[string]int)
	for _, w := range words {
		wordFreq[w]++
	}

	// Calculate importance for target words
	targetSet := make(map[string]bool)
	for _, w := range targetWords {
		targetSet[strings.ToLower(w)] = true
	}

	for word, count := range wordFreq {
		if targetSet[word] {
			// TF-IDF style importance: freq * inverse doc frequency approximation
			importance[word] = float64(count) / totalWords * 10.0
		}
	}

	return importance
}

// Helper to count runes in a string
func runeCount(s string) int {
	return utf8.RuneCountInString(s)
}

// SortedVocab returns vocabulary sorted by ID
func (t *Tokenizer) SortedVocab() []struct {
	Token string
	ID    int32
} {
	result := make([]struct {
		Token string
		ID    int32
	}, 0, len(t.vocab))

	for token, id := range t.vocab {
		result = append(result, struct {
			Token string
			ID    int32
		}{token, id})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}
