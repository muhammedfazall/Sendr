package services

import (
	"context"
	"errors"
	"time"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/pkg/helpers"
)



func errNotFound(name string) error  { return errors.New(name + " not found") }
func errRevoked() error              { return errors.New("api key is revoked") }
func errLimit() error                { return errors.New("api key limit reached") }
func errAlreadyOnPlan() error        { return errors.New("you are already on the") }
func errNoPrice() error              { return errors.New("no price configured") }
func errInvalidSig() error           { return errors.New("invalid payment signature") }
func errSignature() error            { return errors.New("signature") }

// ── In-memory mock implementations of all port interfaces ──

type mockUserRepo struct {
	users map[string]*domain.User
	plans map[string]*domain.Plan
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]*domain.User),
		plans: make(map[string]*domain.Plan),
	}
}

func (m *mockUserRepo) Upsert(_ context.Context, googleID, email, name string) (*domain.User, error) {
	for _, u := range m.users {
		if u.GoogleID == googleID || u.Email == email {
			u.Name = name
			return u, nil
		}
	}
	user := &domain.User{
		ID:       "user-" + googleID,
		Email:    email,
		Name:     name,
		GoogleID: googleID,
		PlanID:   "plan-free",
	}
	m.users[user.ID] = user
	if _, ok := m.plans["plan-free"]; !ok {
		m.plans["plan-free"] = &domain.Plan{ID: "plan-free", Name: "free", DailyLimit: 5, MaxAPIKeys: 1, RateWaitSecs: 30, PricePaise: 0}
	}
	return user, nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errNotFound("user")
	}
	return u, nil
}

func (m *mockUserRepo) FindWithPlan(_ context.Context, id string) (*domain.User, *domain.Plan, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil, errNotFound("user")
	}
	p, ok := m.plans[u.PlanID]
	if !ok {
		return nil, nil, errNotFound("plan")
	}
	return u, p, nil
}

func (m *mockUserRepo) UpdateProfile(_ context.Context, userID, name string) (*domain.User, error) {
	u, ok := m.users[userID]
	if !ok {
		return nil, errNotFound("user")
	}
	u.Name = name
	return u, nil
}

func (m *mockUserRepo) UpdatePlan(_ context.Context, userID, planName string) error {
	u, ok := m.users[userID]
	if !ok {
		return errNotFound("user")
	}
	// Find or create plan
	for id, p := range m.plans {
		if p.Name == planName {
			u.PlanID = id
			return nil
		}
	}
	return errNotFound("plan " + planName)
}

type mockAPIKeyRepo struct {
	keys []domain.APIKey
}

func newMockAPIKeyRepo() *mockAPIKeyRepo {
	return &mockAPIKeyRepo{}
}

func (m *mockAPIKeyRepo) Create(_ context.Context, userID, name, prefix, hashedKey string) (*domain.APIKey, error) {
	k := domain.APIKey{
		ID:     "key-" + prefix,
		UserID: userID,
		Name:   name,
		Prefix: prefix,
		Hashed: hashedKey,
	}
	m.keys = append(m.keys, k)
	return &k, nil
}

func (m *mockAPIKeyRepo) ListByUser(_ context.Context, userID string) ([]domain.APIKey, error) {
	var result []domain.APIKey
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *mockAPIKeyRepo) FindByPrefix(_ context.Context, prefix string) (*domain.APIKey, error) {
	for _, k := range m.keys {
		if k.Prefix == prefix {
			return &k, nil
		}
	}
	return nil, errNotFound("api key")
}

func (m *mockAPIKeyRepo) Revoke(_ context.Context, keyID, userID string) error {
	for i, k := range m.keys {
		if k.ID == keyID && k.UserID == userID {
			m.keys[i].Revoked = true
			return nil
		}
	}
	return errNotFound("api key")
}

type mockTokenStore struct {
	sessions    map[string]string
	blacklisted map[string]bool
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{
		sessions:    make(map[string]string),
		blacklisted: make(map[string]bool),
	}
}

func (m *mockTokenStore) Store(_ context.Context, userID, tokenID string, _ time.Duration) error {
	m.sessions[userID] = tokenID
	return nil
}

func (m *mockTokenStore) Validate(_ context.Context, userID, tokenID string) (bool, error) {
	saved, ok := m.sessions[userID]
	if !ok {
		return false, nil
	}
	return saved == tokenID, nil
}

func (m *mockTokenStore) Delete(_ context.Context, userID string) error {
	delete(m.sessions, userID)
	return nil
}

func (m *mockTokenStore) BlacklistAccessToken(_ context.Context, tokenID string, _ time.Duration) error {
	m.blacklisted[tokenID] = true
	return nil
}

func (m *mockTokenStore) IsAccessTokenBlacklisted(_ context.Context, tokenID string) (bool, error) {
	return m.blacklisted[tokenID], nil
}

type mockJobRepo struct {
	jobs       []domain.Job
	enqueueErr error // injectable error from Enqueue
}

func newMockJobRepo() *mockJobRepo {
	return &mockJobRepo{}
}

func (m *mockJobRepo) Enqueue(_ context.Context, userID, apiKeyID string, payload domain.EmailPayload) (*domain.Job, error) {
	if m.enqueueErr != nil {
		return nil, m.enqueueErr
	}
	j := domain.Job{
		ID:        "job-" + userID,
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Payload:   payload,
		Status:    "pending",
		Retries:   0,
		MaxRetries: 3,
	}
	m.jobs = append(m.jobs, j)
	return &j, nil
}

func (m *mockJobRepo) ClaimBatch(_ context.Context, _ int) ([]domain.Job, error) {
	return nil, nil
}

func (m *mockJobRepo) MarkDone(_ context.Context, _ string) error { return nil }

func (m *mockJobRepo) MarkFailed(_ context.Context, _ string, _ time.Duration) error { return nil }

func (m *mockJobRepo) MoveToDLQ(_ context.Context, _ domain.Job, _ string) error { return nil }

func (m *mockJobRepo) ReclaimZombies(_ context.Context) (int64, error) { return 0, nil }

func (m *mockJobRepo) GetByID(_ context.Context, _ string) (*domain.Job, error) { return nil, nil }

func (m *mockJobRepo) ListByUser(_ context.Context, _ string, _ string, _ int, _ int) ([]domain.Job, error) {
	return nil, nil
}

func (m *mockJobRepo) SetProviderMessageID(_ context.Context, _, _ string) error { return nil }

func (m *mockJobRepo) FindByProviderMessageID(_ context.Context, _ string) (*domain.Job, error) {
	return nil, nil
}

type mockPaymentRepo struct {
	payments map[string]*domain.Payment
}

func newMockPaymentRepo() *mockPaymentRepo {
	return &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
}

func (m *mockPaymentRepo) Create(_ context.Context, p *domain.Payment) error {
	m.payments[p.RazorpayOrderID] = p
	return nil
}

func (m *mockPaymentRepo) FindByOrderID(_ context.Context, orderID string) (*domain.Payment, error) {
	p, ok := m.payments[orderID]
	if !ok {
		return nil, errNotFound("payment")
	}
	return p, nil
}

func (m *mockPaymentRepo) MarkPaid(_ context.Context, orderID, paymentID, signature string) error {
	p, ok := m.payments[orderID]
	if !ok {
		return errNotFound("payment")
	}
	p.Status = "paid"
	p.RazorpayPaymentID = &paymentID
	p.RazorpaySignature = &signature
	return nil
}

func (m *mockPaymentRepo) MarkFailed(_ context.Context, orderID string) error {
	p, ok := m.payments[orderID]
	if !ok {
		return errNotFound("payment")
	}
	p.Status = "failed"
	return nil
}

type mockPlanRepo struct {
	plans []domain.Plan
}

func newMockPlanRepo() *mockPlanRepo {
	m := &mockPlanRepo{}
	m.plans = []domain.Plan{
		{ID: "plan-free", Name: "free", DailyLimit: 5, MaxAPIKeys: 1, RateWaitSecs: 30, PricePaise: 0},
		{ID: "plan-pro", Name: "pro", DailyLimit: 10, MaxAPIKeys: 3, RateWaitSecs: 5, PricePaise: 29900},
		{ID: "plan-max", Name: "max", DailyLimit: -1, MaxAPIKeys: -1, RateWaitSecs: 0, PricePaise: 99900},
	}
	return m
}

func (m *mockPlanRepo) FindByName(_ context.Context, name string) (*domain.Plan, error) {
	for _, p := range m.plans {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, errNotFound("plan")
}

func (m *mockPlanRepo) ListAll(_ context.Context) ([]domain.Plan, error) {
	return m.plans, nil
}

type mockRateLimiter struct {
	allow     bool
	remaining int
	count     int
	checkErr  error // injectable error from Check
}

func newMockRateLimiter() *mockRateLimiter {
	return &mockRateLimiter{allow: true, remaining: 5, count: 0}
}

func (m *mockRateLimiter) Check(_ context.Context, _ string, _ int) (bool, int, error) {
	if m.checkErr != nil {
		return false, 0, m.checkErr
	}
	return m.allow, m.remaining, nil
}

func (m *mockRateLimiter) GetCount(_ context.Context, _ string) (int, error) {
	return m.count, nil
}

// mockAPIKeyService implements ports.APIKeyService for the email service to use.
type mockAPIKeyService struct {
	keys  map[string]*domain.APIKey
	users map[string]string // fullKey -> userID
}

func newMockAPIKeyService() *mockAPIKeyService {
	return &mockAPIKeyService{
		keys:  make(map[string]*domain.APIKey),
		users: make(map[string]string),
	}
}

func (m *mockAPIKeyService) addKey(userID string) (string, *domain.APIKey) {
	k, _ := helpers.GenerateAPIKey()
	key := &domain.APIKey{
		ID:     "key-" + k.Prefix,
		UserID: userID,
		Name:   "test-key",
		Prefix: k.Prefix,
		Hashed: k.Hashed,
	}
	m.keys[k.Full] = key
	m.users[k.Full] = userID
	return k.Full, key
}

func (m *mockAPIKeyService) Validate(_ context.Context, fullKey string) (*domain.APIKey, error) {
	k, ok := m.keys[fullKey]
	if !ok {
		return nil, errNotFound("api key")
	}
	if k.Revoked {
		return nil, errRevoked()
	}
	return k, nil
}

func (m *mockAPIKeyService) Create(_ context.Context, userID, name string) (string, *domain.APIKey, error) {
	k, _ := helpers.GenerateAPIKey()
	key := &domain.APIKey{
		ID:     "key-" + k.Prefix,
		UserID: userID,
		Name:   name,
		Prefix: k.Prefix,
		Hashed: k.Hashed,
	}
	m.keys[k.Full] = key
	m.users[k.Full] = userID
	return k.Full, key, nil
}

func (m *mockAPIKeyService) List(_ context.Context, userID string) ([]domain.APIKey, error) {
	var result []domain.APIKey
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (m *mockAPIKeyService) Revoke(_ context.Context, keyID, userID string) error {
	for fullKey, k := range m.keys {
		if k.ID == keyID && k.UserID == userID {
			k.Revoked = true
			m.keys[fullKey] = k
			return nil
		}
	}
	return errNotFound("api key")
}

// ── Helper constructors that wire full service dependencies with mocks ──

type mockDeps struct {
	users    *mockUserRepo
	apikeys  *mockAPIKeyRepo
	tokens   *mockTokenStore
	jobs     *mockJobRepo
	plans    *mockPlanRepo
	payments *mockPaymentRepo
	limiter  *mockRateLimiter
	keySvc   *mockAPIKeyService
}

func newMockDeps() *mockDeps {
	return &mockDeps{
		users:    newMockUserRepo(),
		apikeys:  newMockAPIKeyRepo(),
		tokens:   newMockTokenStore(),
		jobs:     newMockJobRepo(),
		plans:    newMockPlanRepo(),
		payments: newMockPaymentRepo(),
		limiter:  newMockRateLimiter(),
		keySvc:   newMockAPIKeyService(),
	}
}

// addUserWithPlan adds a user with the given plan name to the mock repos.
func (m *mockDeps) addUserWithPlan(planName string) *domain.User {
	plan, _ := m.plans.FindByName(nil, planName)
	user := &domain.User{
		ID:    "user-test-" + planName,
		Email: "test@" + planName + ".com",
		Name:  "Test " + planName,
		PlanID: plan.ID,
	}
	m.users.users[user.ID] = user
	// Register all plans in the user repo's plan map so upgrades work
	allPlans, _ := m.plans.ListAll(nil)
	for i := range allPlans {
		m.users.plans[allPlans[i].ID] = &allPlans[i]
	}
	return user
}
